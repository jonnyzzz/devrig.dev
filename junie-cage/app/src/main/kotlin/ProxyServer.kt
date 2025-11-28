package org.jonnyzzz.ai.app

import io.ktor.client.*
import io.ktor.client.plugins.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.server.application.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import io.ktor.utils.io.*
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.serialization.json.*
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter
import io.ktor.client.engine.cio.CIO as ClientCIO

fun main() {
    ProxyServer().start()
}

/**
 * Reverse Proxy Server with support for HTTP, WebSocket, and SSE
 *
 * Uses Ktor Netty backend for high performance and production-ready proxying.
 *
 * This implementation uses Ktor's native features and follows official patterns from:
 * - https://github.com/ktorio/ktor-samples/tree/main/reverse-proxy
 * - https://github.com/ktorio/ktor-samples/tree/main/reverse-proxy-ws
 */
class ProxyServer {
    private var server: EmbeddedServer<*, *>? = null
    private val client = HttpClient(ClientCIO) {
        expectSuccess = false
        install(HttpTimeout) {
            requestTimeoutMillis = 600_000
            connectTimeoutMillis = 10_000
            socketTimeoutMillis = 600_000
        }
    }
    private val timeFormatter = DateTimeFormatter.ofPattern("HH:mm:ss.SSS")

    fun start() {
        server = embeddedServer(Netty, port = Config.PROXY_LISTEN_PORT, host = "127.0.0.1") {
            routing {
                // Intercept /api/tags to return our model list (Ollama format)
                get("/api/tags") {
                    val models = Config.getAdvertisedModels().map { modelName ->
                        buildJsonObject {
                            put("name", modelName)
                            put("model", modelName)
                            put("modified_at", "2024-01-01T00:00:00Z")
                            put("size", 0L)
                        }
                    }
                    val response = buildJsonObject {
                        put("models", JsonArray(models))
                    }
                    call.respondText(response.toString(), ContentType.Application.Json)
                }

                // Intercept /v1/models and /models to return our model list (OpenAI format)
                get("/v1/models") { respondModelsOpenAI(call) }
                get("/models") { respondModelsOpenAI(call) }

                // Diagnostic endpoint to check backend connectivity
                get("/debug/backends") {
                    val results = Config.MODEL_BACKENDS.map { backend ->
                        val baseUrl = backend.backendUrl.removeSuffix("/v1").removeSuffix("/")
                        val targetUrl = "$baseUrl/v1/responses"
                        val status = try {
                            val response = client.get(targetUrl)
                            "reachable (${response.status})"
                        } catch (e: Exception) {
                            "ERROR: ${e::class.simpleName}: ${e.message}"
                        }
                        buildJsonObject {
                            put("pattern", backend.pattern)
                            put("targetModel", backend.targetModel)
                            put("backendUrl", backend.backendUrl)
                            put("targetUrl", targetUrl)
                            put("useResponsesApi", backend.useResponsesApi)
                            put("status", status)
                        }
                    }
                    val response = buildJsonObject {
                        put("backends", JsonArray(results))
                    }
                    call.respondText(response.toString(), ContentType.Application.Json)
                }
            }

            // Handle chat/completions endpoints
            intercept(ApplicationCallPipeline.Call) {
                val handledRoutes = setOf("/api/tags", "/v1/models", "/models", "/debug/backends")
                if (call.request.uri !in handledRoutes) {
                    proxyChatCompletions(call)
                }
            }
        }

        CoroutineScope(Dispatchers.IO).launch {
            server?.start(wait = false)
            log("Proxy server started on http://127.0.0.1:${Config.PROXY_LISTEN_PORT}")
            log("Model routing rules (first match wins):")
            Config.MODEL_BACKENDS.forEach { backend ->
                log("  pattern='${backend.pattern}' -> target='${backend.targetModel}' @ ${backend.backendUrl}")
            }
        }
    }

    /**
     * Handle chat/completions requests - routes to appropriate backend based on model
     */
    private suspend fun proxyChatCompletions(call: ApplicationCall) {
        val uri = call.request.uri
        val startTime = System.currentTimeMillis()

        // Only handle chat/completions endpoints
        if (uri !in setOf("/v1/chat/completions", "/chat/completions")) {
            call.respondText(
                """{"error": {"message": "Unknown endpoint: $uri", "type": "invalid_request_error"}}""",
                ContentType.Application.Json,
                HttpStatusCode.NotFound
            )
            return
        }

        try {
            val bodyBytes = call.receiveChannel().toByteArray()
            val json = Json.parseToJsonElement(bodyBytes.decodeToString()).jsonObject
            val inputModel = json["model"]?.jsonPrimitive?.content

            if (inputModel == null) {
                call.respondText(
                    """{"error": {"message": "Missing 'model' field", "type": "invalid_request_error"}}""",
                    ContentType.Application.Json,
                    HttpStatusCode.BadRequest
                )
                return
            }

            val match = Config.getBackendForModel(inputModel)
            if (match == null) {
                call.respondText(
                    """{"error": {"message": "Model '$inputModel' not found. Available patterns: ${Config.MODEL_BACKENDS.map { it.pattern }}", "type": "model_not_found"}}""",
                    ContentType.Application.Json,
                    HttpStatusCode.NotFound
                )
                return
            }

            logRequest(call, bodyBytes)

            // Log the model routing decision
            if (match.inputModel != match.targetModel) {
                log("MODEL ROUTING: '$inputModel' matched pattern '${match.backend.pattern}' -> target '${match.targetModel}'")
            } else {
                log("MODEL ROUTING: '$inputModel' matched pattern '${match.backend.pattern}' (exact match)")
            }

            if (match.backend.useResponsesApi) {
                proxyChatCompletionsViaResponsesApi(call, json, match, startTime)
            } else {
                proxyDirectChatCompletions(call, bodyBytes, match, startTime)
            }

        } catch (e: Exception) {
            val duration = System.currentTimeMillis() - startTime
            log("ERROR: ${call.request.httpMethod.value} $uri -> ${e.message} (${duration}ms)")
            call.respondText(
                """{"error": {"message": "Proxy error: ${e.message}", "type": "proxy_error"}}""",
                ContentType.Application.Json,
                HttpStatusCode.BadGateway
            )
        }
    }

    /**
     * Proxy chat/completions directly (without Responses API conversion)
     * For OpenAI-compatible backends like Ollama
     */
    private suspend fun proxyDirectChatCompletions(
        call: ApplicationCall,
        bodyBytes: ByteArray,
        match: ModelMatch,
        startTime: Long
    ) {
        val backend = match.backend
        // Build URL - use /v1/chat/completions for OpenAI compatibility
        val baseUrl = backend.backendUrl.removeSuffix("/v1").removeSuffix("/")
        val targetUrl = "$baseUrl/v1/chat/completions"

        // Replace model in request body with target model
        val modifiedBody = if (match.inputModel != match.targetModel) {
            val json = Json.parseToJsonElement(bodyBytes.decodeToString()).jsonObject.toMutableMap()
            json["model"] = JsonPrimitive(match.targetModel)
            JsonObject(json).toString().toByteArray()
        } else {
            bodyBytes
        }

        log("POST ${call.request.uri} [input: ${match.inputModel}] [target: ${match.targetModel}] [pattern: ${backend.pattern}] -> $targetUrl [direct proxy]")

        try {
            client.preparePost(targetUrl) {
                contentType(ContentType.Application.Json)
                setBody(modifiedBody)
                call.request.headers[HttpHeaders.Authorization]?.let {
                    header(HttpHeaders.Authorization, it)
                }
            }.execute { response ->
                call.respond(object : io.ktor.http.content.OutgoingContent.WriteChannelContent() {
                    override val status = response.status
                    override val contentType = response.contentType()

                    override suspend fun writeTo(channel: ByteWriteChannel) {
                        val responseBody = response.bodyAsChannel()
                        val buffer = ByteArray(256)

                        while (!responseBody.isClosedForRead) {
                            if (responseBody.availableForRead == 0) {
                                if (!responseBody.awaitContent()) break
                            }
                            val bytesRead = responseBody.readAvailable(buffer, 0, buffer.size)
                            if (bytesRead <= 0) break
                            channel.writeFully(buffer, 0, bytesRead)
                            channel.flush()
                        }
                    }
                })

                val duration = System.currentTimeMillis() - startTime
                log("POST ${call.request.uri} <- ${response.status.value} (${duration}ms)")
            }
        } catch (e: Exception) {
            val duration = System.currentTimeMillis() - startTime
            log("ERROR: POST ${call.request.uri} -> ${e::class.simpleName}: ${e.message} (${duration}ms)")
            log("  Target URL was: $targetUrl")
            e.cause?.let { log("  Caused by: ${it::class.simpleName}: ${it.message}") }
            call.respondText(
                """{"error": {"message": "Proxy error: ${e.message}", "type": "proxy_error", "target": "$targetUrl"}}""",
                ContentType.Application.Json,
                HttpStatusCode.BadGateway
            )
        }
    }

    /**
     * Convert Chat Completions request to Responses API, proxy it, and convert response back
     */
    private suspend fun proxyChatCompletionsViaResponsesApi(
        call: ApplicationCall,
        originalJson: JsonObject,
        match: ModelMatch,
        startTime: Long
    ) {
        val backend = match.backend
        val isStream = originalJson["stream"]?.jsonPrimitive?.boolean ?: false

        // Convert Chat Completions -> Responses API request (using target model)
        val responsesApiBody = buildJsonObject {
            put("model", match.targetModel)
            originalJson["messages"]?.let { put("input", it) }
            if (isStream) put("stream", true)
            originalJson["max_tokens"]?.let { put("max_output_tokens", it) }
            originalJson["temperature"]?.let { put("temperature", it) }
            originalJson["top_p"]?.let { put("top_p", it) }
        }

        // Build responses API URL - strip /v1 suffix if present since we add /v1/responses
        val baseUrl = backend.backendUrl.removeSuffix("/v1").removeSuffix("/")
        val targetUrl = "$baseUrl/v1/responses"
        log("POST ${call.request.uri} [input: ${match.inputModel}] [target: ${match.targetModel}] [pattern: ${backend.pattern}] -> $targetUrl [Responses API conversion]")

        try {
            client.preparePost(targetUrl) {
                contentType(ContentType.Application.Json)
                setBody(responsesApiBody.toString())
                // Copy auth headers
                call.request.headers[HttpHeaders.Authorization]?.let {
                    header(HttpHeaders.Authorization, it)
                }
            }.execute { response ->
                // Return the input model name to the client (they asked for it)
                val responseModel = match.inputModel
                if (!isStream) {
                    // Non-streaming: convert response and return
                    val responseText = response.bodyAsChannel().toByteArray().decodeToString()
                    val chatResponse = convertResponsesApiToChatCompletions(responseText, responseModel)
                    call.respondText(chatResponse, ContentType.Application.Json, response.status)
                } else {
                    // Streaming: convert SSE events on the fly
                    call.respond(object : io.ktor.http.content.OutgoingContent.WriteChannelContent() {
                        override val status = response.status
                        override val contentType = ContentType.Text.EventStream

                        override suspend fun writeTo(channel: ByteWriteChannel) {
                            val responseBody = response.bodyAsChannel()
                            val chatId = "chatcmpl-${System.currentTimeMillis()}"
                            var chunkIndex = 0

                            // Read line by line for SSE parsing
                            while (!responseBody.isClosedForRead) {
                                val line = responseBody.readUTF8Line() ?: break

                                if (line.startsWith("data:")) {
                                    val data = line.substring(5).trim()
                                    if (data.isEmpty()) continue

                                    val converted = convertResponsesApiEventToChatCompletions(data, chatId, responseModel, chunkIndex++)
                                    if (converted != null) {
                                        channel.writeStringUtf8("data: $converted\n\n")
                                        channel.flush()
                                    }
                                }
                            }
                            // Send [DONE]
                            channel.writeStringUtf8("data: [DONE]\n\n")
                            channel.flush()
                        }
                    })
                }
                val duration = System.currentTimeMillis() - startTime
                log("POST /v1/chat/completions <- ${response.status.value} (${duration}ms)")
            }
        } catch (e: Exception) {
            val duration = System.currentTimeMillis() - startTime
            log("ERROR: POST ${call.request.uri} -> ${e::class.simpleName}: ${e.message} (${duration}ms)")
            log("  Target URL was: $targetUrl")
            e.cause?.let { log("  Caused by: ${it::class.simpleName}: ${it.message}") }
            call.respondText(
                """{"error": {"message": "Proxy error: ${e.message}", "type": "proxy_error", "target": "$targetUrl"}}""",
                ContentType.Application.Json,
                HttpStatusCode.BadGateway
            )
        }
    }

    private fun convertResponsesApiToChatCompletions(responseText: String, model: String): String {
        return try {
            val json = Json.parseToJsonElement(responseText).jsonObject
            val output = json["output"]?.jsonArray?.firstOrNull()?.jsonObject
            val content = output?.get("content")?.jsonArray?.firstOrNull()?.jsonObject
                ?.get("text")?.jsonPrimitive?.content ?: ""

            buildJsonObject {
                put("id", "chatcmpl-${System.currentTimeMillis()}")
                put("object", "chat.completion")
                put("created", System.currentTimeMillis() / 1000)
                put("model", model)
                put("choices", buildJsonArray {
                    add(buildJsonObject {
                        put("index", 0)
                        put("message", buildJsonObject {
                            put("role", "assistant")
                            put("content", content)
                        })
                        put("finish_reason", "stop")
                    })
                })
            }.toString()
        } catch (e: Exception) {
            log("Error converting response: ${e.message}")
            responseText
        }
    }

    private fun convertResponsesApiEventToChatCompletions(data: String, chatId: String, model: String, index: Int): String? {
        return try {
            val json = Json.parseToJsonElement(data).jsonObject
            val type = json["type"]?.jsonPrimitive?.content

            when (type) {
                "response.output_text.delta" -> {
                    val delta = json["delta"]?.jsonPrimitive?.content ?: return null
                    buildJsonObject {
                        put("id", chatId)
                        put("object", "chat.completion.chunk")
                        put("created", System.currentTimeMillis() / 1000)
                        put("model", model)
                        put("choices", buildJsonArray {
                            add(buildJsonObject {
                                put("index", 0)
                                put("delta", buildJsonObject { put("content", delta) })
                                put("finish_reason", JsonNull)
                            })
                        })
                    }.toString()
                }
                "response.completed" -> {
                    buildJsonObject {
                        put("id", chatId)
                        put("object", "chat.completion.chunk")
                        put("created", System.currentTimeMillis() / 1000)
                        put("model", model)
                        put("choices", buildJsonArray {
                            add(buildJsonObject {
                                put("index", 0)
                                put("delta", buildJsonObject { })
                                put("finish_reason", "stop")
                            })
                        })
                    }.toString()
                }
                else -> null
            }
        } catch (e: Exception) {
            null
        }
    }

    private fun extractSSEContent(chunk: String): String? {
        // Extract content from SSE data lines
        // Format: data: {"choices":[{"delta":{"content":"text"}}]}
        // or Ollama: data: {"message":{"content":"text"}}
        return try {
            chunk.lines()
                .filter { it.startsWith("data:") && !it.contains("[DONE]") }
                .mapNotNull { line ->
                    val jsonText = line.substring(5).trim()
                    if (jsonText.isEmpty()) return@mapNotNull null

                    val json = Json.parseToJsonElement(jsonText).jsonObject

                    // OpenAI format
                    json["choices"]?.jsonArray?.firstOrNull()?.jsonObject
                        ?.get("delta")?.jsonObject
                        ?.get("content")?.jsonPrimitive?.content
                        ?: // Ollama format
                        json["message"]?.jsonObject
                            ?.get("content")?.jsonPrimitive?.content
                }
                .firstOrNull()
        } catch (e: Exception) {
            null
        }
    }

    private fun logRequest(call: ApplicationCall, bodyBytes: ByteArray?) {
        val method = call.request.httpMethod.value
        val uri = call.request.uri

        // Check if this is an OpenAI-compatible or Ollama API request
        if (bodyBytes != null && (uri.contains("/v1/chat/completions") || uri.contains("/api/chat"))) {
            try {
                val bodyText = bodyBytes.decodeToString()
                val json = Json.parseToJsonElement(bodyText).jsonObject
                val messages = json["messages"]?.jsonArray
                val prompt = json["prompt"]?.jsonPrimitive?.content
                val stream = json["stream"]?.jsonPrimitive?.boolean ?: false

                val streamIndicator = if (stream) " [SSE]" else ""

                if (messages != null) {
                    val lastMessage = messages.lastOrNull()?.jsonObject
                    val content = lastMessage?.get("content")?.jsonPrimitive?.content
                    if (content != null) {
                        val preview = content.replace("\n", " ").replace("\r", " ")
                            .let { if (it.length > 300) it.take(300) + "..." else it }
                        log("$method $uri$streamIndicator [prompt: $preview]")
                        return
                    }
                } else if (prompt != null) {
                    val preview = prompt.replace("\n", " ").replace("\r", " ")
                        .let { if (it.length > 300) it.take(300) + "..." else it }
                    log("$method $uri$streamIndicator [prompt: $preview]")
                    return
                }
            } catch (e: Exception) {
                // If parsing fails, just log normally
            }
        }

        log("$method $uri")
    }

    private fun log(message: String) {
        val timestamp = LocalDateTime.now().format(timeFormatter)
        println("[$timestamp] [PROXY] $message")
    }

    private suspend fun respondModelsOpenAI(call: ApplicationCall) {
        val models = Config.getAdvertisedModels().map { modelName ->
            buildJsonObject {
                put("id", modelName)
                put("object", "model")
                put("created", 1700000000L)
                put("owned_by", "organization")
            }
        }
        val response = buildJsonObject {
            put("object", "list")
            put("data", JsonArray(models))
        }
        call.respondText(response.toString(), ContentType.Application.Json)
    }

    fun stop() {
        server?.stop(1000, 2000)
        client.close()
    }
}
