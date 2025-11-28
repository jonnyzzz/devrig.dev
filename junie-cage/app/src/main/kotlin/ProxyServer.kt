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
            // Log all incoming requests and responses
            intercept(ApplicationCallPipeline.Monitoring) {
                val method = call.request.httpMethod.value
                val uri = call.request.uri
                val remoteHost = call.request.local.remoteHost
                val contentType = call.request.contentType().toString()
                val contentLength = call.request.headers["Content-Length"] ?: "?"
                log(">>> $method $uri (from $remoteHost) [Content-Type: $contentType, Content-Length: $contentLength]")

                // Continue processing, then log response
                proceed()

                // Log response status
                val status = call.response.status()
                if (status != null) {
                    if (status.value >= 400) {
                        log("<<< $method $uri -> ${status.value} ${status.description}")
                    }
                }
            }

            routing {
                // Health check endpoint - support GET, HEAD, OPTIONS
                get("/") {
                    call.respondText("OK", ContentType.Text.Plain)
                }
                head("/") {
                    call.respond(HttpStatusCode.OK)
                }
                options("/") {
                    call.response.header("Allow", "GET, HEAD, OPTIONS, POST")
                    call.respond(HttpStatusCode.OK)
                }
                // Debug: capture POST / requests to see what's being sent
                post("/") {
                    val bodyBytes = call.receiveChannel().toByteArray()
                    val bodyText = bodyBytes.decodeToString()
                    log("!!! POST / received body (${bodyBytes.size} bytes): ${truncateForLog(bodyText, 1000)}")

                    // Try to parse as JSON and extract useful info
                    try {
                        val json = Json.parseToJsonElement(bodyText).jsonObject
                        val model = json["model"]?.jsonPrimitive?.content
                        val method = json["method"]?.jsonPrimitive?.content
                        val endpoint = json["endpoint"]?.jsonPrimitive?.content
                        log("!!! POST / parsed: model=$model, method=$method, endpoint=$endpoint")
                        log("!!! POST / keys: ${json.keys}")
                    } catch (e: Exception) {
                        log("!!! POST / not valid JSON: ${e.message}")
                    }

                    // Return error with helpful message
                    call.respondText(
                        """{"error": {"message": "POST to / is not supported. Use /v1/chat/completions or /v1/responses", "type": "invalid_endpoint"}}""",
                        ContentType.Application.Json,
                        HttpStatusCode.BadRequest
                    )
                }

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

            // Handle chat/completions, responses, and Ollama chat endpoints
            intercept(ApplicationCallPipeline.Call) {
                val handledRoutes = setOf("/", "/api/tags", "/v1/models", "/models", "/debug/backends")
                if (call.request.uri !in handledRoutes) {
                    when (call.request.uri) {
                        "/v1/responses", "/responses" -> proxyResponsesApi(call)
                        "/api/chat" -> proxyOllamaChat(call)
                        else -> proxyChatCompletions(call)
                    }
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
            log("!!! UNHANDLED ENDPOINT: ${call.request.httpMethod.value} $uri")
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
     * Handle Ollama /api/chat requests - proxy directly to Ollama backend
     */
    private suspend fun proxyOllamaChat(call: ApplicationCall) {
        val uri = call.request.uri
        val startTime = System.currentTimeMillis()

        try {
            val bodyBytes = call.receiveChannel().toByteArray()
            val json = Json.parseToJsonElement(bodyBytes.decodeToString()).jsonObject
            val inputModel = json["model"]?.jsonPrimitive?.content

            if (inputModel == null) {
                call.respondText(
                    """{"error": "Missing 'model' field"}""",
                    ContentType.Application.Json,
                    HttpStatusCode.BadRequest
                )
                return
            }

            val match = Config.getBackendForModel(inputModel)
            if (match == null) {
                call.respondText(
                    """{"error": "Model '$inputModel' not found"}""",
                    ContentType.Application.Json,
                    HttpStatusCode.NotFound
                )
                return
            }

            // Log prompt
            extractPromptFromMessages(json)?.let { prompt ->
                log(">>> PROMPT [Ollama]: ${truncateForLog(prompt)}")
            }

            log("MODEL ROUTING [Ollama]: '$inputModel' matched pattern '${match.backend.pattern}' -> target '${match.targetModel}'")

            // Replace model in request body with target model
            val modifiedBody = if (match.inputModel != match.targetModel) {
                val jsonMap = json.toMutableMap()
                jsonMap["model"] = JsonPrimitive(match.targetModel)
                JsonObject(jsonMap).toString().toByteArray()
            } else {
                bodyBytes
            }

            // Build Ollama API URL
            val baseUrl = match.backend.backendUrl.removeSuffix("/v1").removeSuffix("/")
            val targetUrl = "$baseUrl/api/chat"
            val isStream = json["stream"]?.jsonPrimitive?.boolean ?: true // Ollama defaults to streaming

            log("POST $uri [input: ${match.inputModel}] [target: ${match.targetModel}] -> $targetUrl [Ollama chat]")

            client.preparePost(targetUrl) {
                contentType(ContentType.Application.Json)
                setBody(modifiedBody)
            }.execute { response ->
                val accumulatedResponse = StringBuilder()

                call.respond(object : io.ktor.http.content.OutgoingContent.WriteChannelContent() {
                    override val status = response.status
                    override val contentType = response.contentType()

                    override suspend fun writeTo(channel: ByteWriteChannel) {
                        val responseBody = response.bodyAsChannel()
                        val buffer = ByteArray(256)
                        val lineBuffer = StringBuilder()

                        while (!responseBody.isClosedForRead) {
                            if (responseBody.availableForRead == 0) {
                                if (!responseBody.awaitContent()) break
                            }
                            val bytesRead = responseBody.readAvailable(buffer, 0, buffer.size)
                            if (bytesRead <= 0) break

                            // Track content for logging (Ollama streams JSON objects line by line)
                            if (isStream) {
                                val chunk = buffer.decodeToString(0, bytesRead)
                                lineBuffer.append(chunk)
                                while (lineBuffer.contains("\n")) {
                                    val lineEnd = lineBuffer.indexOf("\n")
                                    val line = lineBuffer.substring(0, lineEnd).trim()
                                    lineBuffer.delete(0, lineEnd + 1)
                                    if (line.isNotEmpty()) {
                                        try {
                                            val lineJson = Json.parseToJsonElement(line).jsonObject
                                            val content = lineJson["message"]?.jsonObject?.get("content")?.jsonPrimitive?.content
                                            content?.let { accumulatedResponse.append(it) }
                                        } catch (_: Exception) {}
                                    }
                                }
                            }

                            channel.writeFully(buffer, 0, bytesRead)
                            channel.flush()
                        }
                    }
                })

                val duration = System.currentTimeMillis() - startTime
                if (accumulatedResponse.isNotEmpty()) {
                    log("<<< RESPONSE [Ollama]: ${truncateForLog(accumulatedResponse.toString())}")
                }
                log("POST $uri <- ${response.status.value} (${duration}ms)")
            }

        } catch (e: Exception) {
            val duration = System.currentTimeMillis() - startTime
            log("ERROR: ${call.request.httpMethod.value} $uri -> ${e.message} (${duration}ms)")
            call.respondText(
                """{"error": "Proxy error: ${e.message}"}""",
                ContentType.Application.Json,
                HttpStatusCode.BadGateway
            )
        }
    }

    /**
     * Handle Responses API requests - convert to Chat Completions for OpenAI-compatible backends
     */
    private suspend fun proxyResponsesApi(call: ApplicationCall) {
        val uri = call.request.uri
        val startTime = System.currentTimeMillis()

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
                    """{"error": {"message": "Model '$inputModel' not found", "type": "model_not_found"}}""",
                    ContentType.Application.Json,
                    HttpStatusCode.NotFound
                )
                return
            }

            log("MODEL ROUTING [Responses API]: '$inputModel' matched pattern '${match.backend.pattern}' -> target '${match.targetModel}'")

            // Log prompt
            extractPromptFromResponsesApi(json)?.let { prompt ->
                log(">>> PROMPT: ${truncateForLog(prompt)}")
            }

            if (match.backend.useResponsesApi) {
                // Backend supports Responses API - proxy directly
                proxyDirectResponsesApi(call, bodyBytes, match, startTime)
            } else {
                // Backend uses OpenAI API - convert Responses API to Chat Completions
                proxyResponsesApiViaChatCompletions(call, json, match, startTime)
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
     * Proxy Responses API directly to backend that supports it
     */
    private suspend fun proxyDirectResponsesApi(
        call: ApplicationCall,
        bodyBytes: ByteArray,
        match: ModelMatch,
        startTime: Long
    ) {
        val backend = match.backend
        val baseUrl = backend.backendUrl.removeSuffix("/v1").removeSuffix("/")
        val targetUrl = "$baseUrl/v1/responses"

        // Replace model in request body with target model
        val modifiedBody = if (match.inputModel != match.targetModel) {
            val json = Json.parseToJsonElement(bodyBytes.decodeToString()).jsonObject.toMutableMap()
            json["model"] = JsonPrimitive(match.targetModel)
            JsonObject(json).toString().toByteArray()
        } else {
            bodyBytes
        }

        log("POST ${call.request.uri} [input: ${match.inputModel}] [target: ${match.targetModel}] -> $targetUrl [direct Responses API]")

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
            call.respondText(
                """{"error": {"message": "Proxy error: ${e.message}", "type": "proxy_error", "target": "$targetUrl"}}""",
                ContentType.Application.Json,
                HttpStatusCode.BadGateway
            )
        }
    }

    /**
     * Convert Responses API request to Chat Completions, proxy it, and convert response back
     */
    private suspend fun proxyResponsesApiViaChatCompletions(
        call: ApplicationCall,
        originalJson: JsonObject,
        match: ModelMatch,
        startTime: Long
    ) {
        val backend = match.backend
        val isStream = originalJson["stream"]?.jsonPrimitive?.boolean ?: false

        // Convert Responses API -> Chat Completions request
        val chatCompletionsBody = buildJsonObject {
            put("model", match.targetModel)
            // Convert input to messages format
            val input = originalJson["input"]
            when {
                input is JsonArray -> put("messages", input)
                input is JsonPrimitive && input.isString -> {
                    put("messages", buildJsonArray {
                        add(buildJsonObject {
                            put("role", "user")
                            put("content", input.content)
                        })
                    })
                }
                else -> originalJson["input"]?.let { put("messages", it) }
            }
            if (isStream) put("stream", true)
            originalJson["max_output_tokens"]?.let { put("max_tokens", it) }
            originalJson["temperature"]?.let { put("temperature", it) }
            originalJson["top_p"]?.let { put("top_p", it) }
        }

        val baseUrl = backend.backendUrl.removeSuffix("/v1").removeSuffix("/")
        val targetUrl = "$baseUrl/v1/chat/completions"
        log("POST ${call.request.uri} [input: ${match.inputModel}] [target: ${match.targetModel}] -> $targetUrl [Responses->ChatCompletions conversion]")

        try {
            client.preparePost(targetUrl) {
                contentType(ContentType.Application.Json)
                setBody(chatCompletionsBody.toString())
                call.request.headers[HttpHeaders.Authorization]?.let {
                    header(HttpHeaders.Authorization, it)
                }
            }.execute { response ->
                val responseModel = match.inputModel
                val accumulatedResponse = StringBuilder()

                if (!isStream) {
                    // Non-streaming: convert response back to Responses API format
                    val responseText = response.bodyAsChannel().toByteArray().decodeToString()
                    // Extract content for logging
                    try {
                        val json = Json.parseToJsonElement(responseText).jsonObject
                        val content = json["choices"]?.jsonArray?.firstOrNull()?.jsonObject
                            ?.get("message")?.jsonObject?.get("content")?.jsonPrimitive?.content
                        content?.let { accumulatedResponse.append(it) }
                    } catch (_: Exception) {}
                    val responsesApiResponse = convertChatCompletionsToResponsesApi(responseText, responseModel)
                    call.respondText(responsesApiResponse, ContentType.Application.Json, response.status)
                } else {
                    // Streaming: convert SSE events on the fly
                    call.respond(object : io.ktor.http.content.OutgoingContent.WriteChannelContent() {
                        override val status = response.status
                        override val contentType = ContentType.Text.EventStream

                        override suspend fun writeTo(channel: ByteWriteChannel) {
                            val responseBody = response.bodyAsChannel()
                            val responseId = "resp-${System.currentTimeMillis()}"

                            while (!responseBody.isClosedForRead) {
                                val line = responseBody.readUTF8Line() ?: break

                                if (line.startsWith("data:")) {
                                    val data = line.substring(5).trim()
                                    if (data.isEmpty() || data == "[DONE]") continue

                                    // Extract delta for logging
                                    try {
                                        val json = Json.parseToJsonElement(data).jsonObject
                                        val delta = json["choices"]?.jsonArray?.firstOrNull()?.jsonObject
                                            ?.get("delta")?.jsonObject?.get("content")?.jsonPrimitive?.content
                                        delta?.let { accumulatedResponse.append(it) }
                                    } catch (_: Exception) {}

                                    val converted = convertChatCompletionsEventToResponsesApi(data, responseId, responseModel)
                                    if (converted != null) {
                                        channel.writeStringUtf8("data: $converted\n\n")
                                        channel.flush()
                                    }
                                }
                            }
                            // Send completed event
                            val completedEvent = buildJsonObject {
                                put("type", "response.completed")
                                put("response", buildJsonObject {
                                    put("id", responseId)
                                    put("model", responseModel)
                                    put("status", "completed")
                                })
                            }
                            channel.writeStringUtf8("data: $completedEvent\n\n")
                            channel.flush()
                        }
                    })
                }
                val duration = System.currentTimeMillis() - startTime
                if (accumulatedResponse.isNotEmpty()) {
                    log("<<< RESPONSE: ${truncateForLog(accumulatedResponse.toString())}")
                }
                log("POST ${call.request.uri} <- ${response.status.value} (${duration}ms)")
            }
        } catch (e: Exception) {
            val duration = System.currentTimeMillis() - startTime
            log("ERROR: POST ${call.request.uri} -> ${e::class.simpleName}: ${e.message} (${duration}ms)")
            call.respondText(
                """{"error": {"message": "Proxy error: ${e.message}", "type": "proxy_error", "target": "$targetUrl"}}""",
                ContentType.Application.Json,
                HttpStatusCode.BadGateway
            )
        }
    }

    private fun convertChatCompletionsToResponsesApi(responseText: String, model: String): String {
        return try {
            val json = Json.parseToJsonElement(responseText).jsonObject
            val choice = json["choices"]?.jsonArray?.firstOrNull()?.jsonObject
            val content = choice?.get("message")?.jsonObject?.get("content")?.jsonPrimitive?.content ?: ""

            buildJsonObject {
                put("id", "resp-${System.currentTimeMillis()}")
                put("object", "response")
                put("created_at", System.currentTimeMillis() / 1000)
                put("model", model)
                put("status", "completed")
                put("output", buildJsonArray {
                    add(buildJsonObject {
                        put("type", "message")
                        put("role", "assistant")
                        put("content", buildJsonArray {
                            add(buildJsonObject {
                                put("type", "output_text")
                                put("text", content)
                            })
                        })
                    })
                })
            }.toString()
        } catch (e: Exception) {
            log("Error converting chat completions response to Responses API: ${e.message}")
            responseText
        }
    }

    private fun convertChatCompletionsEventToResponsesApi(data: String, responseId: String, model: String): String? {
        return try {
            val json = Json.parseToJsonElement(data).jsonObject
            val delta = json["choices"]?.jsonArray?.firstOrNull()?.jsonObject
                ?.get("delta")?.jsonObject?.get("content")?.jsonPrimitive?.content

            if (delta != null) {
                buildJsonObject {
                    put("type", "response.output_text.delta")
                    put("delta", delta)
                }.toString()
            } else {
                null
            }
        } catch (e: Exception) {
            null
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

        // Extract and log prompt
        val promptJson = Json.parseToJsonElement(bodyBytes.decodeToString()).jsonObject
        val isStream = promptJson["stream"]?.jsonPrimitive?.boolean ?: false
        extractPromptFromMessages(promptJson)?.let { prompt ->
            log(">>> PROMPT: ${truncateForLog(prompt)}")
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
                val accumulatedResponse = StringBuilder()

                call.respond(object : io.ktor.http.content.OutgoingContent.WriteChannelContent() {
                    override val status = response.status
                    override val contentType = response.contentType()

                    override suspend fun writeTo(channel: ByteWriteChannel) {
                        val responseBody = response.bodyAsChannel()
                        val buffer = ByteArray(256)
                        val lineBuffer = StringBuilder()

                        while (!responseBody.isClosedForRead) {
                            if (responseBody.availableForRead == 0) {
                                if (!responseBody.awaitContent()) break
                            }
                            val bytesRead = responseBody.readAvailable(buffer, 0, buffer.size)
                            if (bytesRead <= 0) break

                            // Track content for logging
                            if (isStream) {
                                val chunk = buffer.decodeToString(0, bytesRead)
                                lineBuffer.append(chunk)
                                // Extract SSE content from complete lines
                                while (lineBuffer.contains("\n")) {
                                    val lineEnd = lineBuffer.indexOf("\n")
                                    val line = lineBuffer.substring(0, lineEnd)
                                    lineBuffer.delete(0, lineEnd + 1)
                                    extractSSEContent(line)?.let { accumulatedResponse.append(it) }
                                }
                            }

                            channel.writeFully(buffer, 0, bytesRead)
                            channel.flush()
                        }
                    }
                })

                val duration = System.currentTimeMillis() - startTime
                if (isStream && accumulatedResponse.isNotEmpty()) {
                    log("<<< RESPONSE: ${truncateForLog(accumulatedResponse.toString())}")
                }
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

        // Log prompt
        extractPromptFromMessages(originalJson)?.let { prompt ->
            log(">>> PROMPT: ${truncateForLog(prompt)}")
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
                val accumulatedResponse = StringBuilder()

                if (!isStream) {
                    // Non-streaming: convert response and return
                    val responseText = response.bodyAsChannel().toByteArray().decodeToString()
                    // Extract content for logging
                    try {
                        val json = Json.parseToJsonElement(responseText).jsonObject
                        val content = json["output"]?.jsonArray?.firstOrNull()?.jsonObject
                            ?.get("content")?.jsonArray?.firstOrNull()?.jsonObject
                            ?.get("text")?.jsonPrimitive?.content
                        content?.let { accumulatedResponse.append(it) }
                    } catch (_: Exception) {}
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

                                    // Extract delta for logging
                                    try {
                                        val json = Json.parseToJsonElement(data).jsonObject
                                        if (json["type"]?.jsonPrimitive?.content == "response.output_text.delta") {
                                            json["delta"]?.jsonPrimitive?.content?.let { accumulatedResponse.append(it) }
                                        }
                                    } catch (_: Exception) {}

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
                if (accumulatedResponse.isNotEmpty()) {
                    log("<<< RESPONSE: ${truncateForLog(accumulatedResponse.toString())}")
                }
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
                        log("$method $uri$streamIndicator [prompt: ${truncateForLog(content)}]")
                        return
                    }
                } else if (prompt != null) {
                    log("$method $uri$streamIndicator [prompt: ${truncateForLog(prompt)}]")
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

    private fun truncateForLog(text: String, maxLen: Int = 500): String {
        val oneLine = text.replace("\n", " ").replace("\r", " ").replace("\\s+".toRegex(), " ").trim()
        return if (oneLine.length > maxLen) oneLine.take(maxLen) + "..." else oneLine
    }

    private fun extractPromptFromMessages(json: JsonObject): String? {
        val messages = json["messages"]?.jsonArray ?: return null
        val lastMessage = messages.lastOrNull()?.jsonObject ?: return null
        return lastMessage["content"]?.jsonPrimitive?.content
    }

    private fun extractPromptFromResponsesApi(json: JsonObject): String? {
        val input = json["input"] ?: return null
        return when {
            input is JsonPrimitive && input.isString -> input.content
            input is JsonArray -> {
                val lastMessage = input.lastOrNull()?.jsonObject
                lastMessage?.get("content")?.jsonPrimitive?.content
            }
            else -> null
        }
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
