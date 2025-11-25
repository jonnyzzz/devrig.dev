package org.jonnyzzz.ai.app

import io.ktor.client.*
import io.ktor.client.plugins.*
import io.ktor.client.plugins.websocket.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.http.Url
import io.ktor.server.application.*
import io.ktor.server.engine.*
import io.ktor.server.netty.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import io.ktor.server.websocket.*
import io.ktor.server.websocket.WebSockets
import io.ktor.utils.io.*
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.launch
import kotlinx.serialization.json.*
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter
import io.ktor.client.engine.cio.CIO as ClientCIO
import io.ktor.client.plugins.websocket.WebSockets as ClientWebSockets

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

        // Install WebSocket support for client
        install(ClientWebSockets)

        // Configure timeout using Ktor's HttpTimeout plugin
        install(HttpTimeout) {
            requestTimeoutMillis = 600_000 // 10 minutes for slow Ollama responses
            connectTimeoutMillis = 10_000  // 10 seconds to establish connection
            socketTimeoutMillis = 600_000  // 10 minutes for socket operations
        }
    }
    private val timeFormatter = DateTimeFormatter.ofPattern("HH:mm:ss.SSS")

    fun start() {
        server = embeddedServer(Netty, port = Config.PROXY_LISTEN_PORT, host = "127.0.0.1") {
            // Install WebSocket support for server
            install(WebSockets)

            // Use routing to handle WebSocket and regular HTTP requests
            routing {
                // WebSocket proxy - catches all WebSocket upgrade requests
                webSocket("{...}") {
                    proxyWebSocketSession(call)
                }
            }

            // Use Ktor's native intercept pattern for regular HTTP/SSE proxying
            // This intercepts all non-WebSocket requests at the pipeline level
            intercept(ApplicationCallPipeline.Call) {
                // Only proxy if not handled by routing (i.e., not WebSocket)
                if (call.request.header(HttpHeaders.Upgrade)?.lowercase() != "websocket") {
                    proxyHttpRequest(call)
                }
            }
        }

        CoroutineScope(Dispatchers.IO).launch {
            server?.start(wait = false)
            log("Proxy server started on http://127.0.0.1:${Config.PROXY_LISTEN_PORT}")
            log("Forwarding to ${Config.TARGET_BASE_URL}")
            log("Supports: HTTP, WebSocket (WS), Server-Sent Events (SSE)")
        }
    }

    /**
     * Proxy HTTP requests including SSE streaming
     * Uses Ktor's native channel-based streaming for efficient forwarding
     */
    private suspend fun proxyHttpRequest(call: ApplicationCall) {
        val targetUrl = Config.buildHttpUrl(call.request.uri)
        val startTime = System.currentTimeMillis()

        try {
            // Read body for logging (if present)
            val bodyBytes = if (call.request.httpMethod in listOf(HttpMethod.Post, HttpMethod.Put, HttpMethod.Patch)) {
                call.receiveChannel().toByteArray()
            } else {
                null
            }

            // Log request with prompt preview for OpenAI API
            logRequest(call, bodyBytes)

            // Use Ktor's HttpStatement for streaming responses without buffering
            // This is critical for SSE (Server-Sent Events) streaming from OpenAI-compatible APIs
            client.prepareRequest(targetUrl) {
                method = call.request.httpMethod

                // Copy headers using Ktor's native header handling (excluding hop-by-hop headers)
                headers {
                    call.request.headers.forEach { name, values ->
                        // Exclude hop-by-hop headers as per RFC 2616
                        if (!name.equals(HttpHeaders.Host, ignoreCase = true) &&
                            !name.equals(HttpHeaders.Connection, ignoreCase = true) &&
                            !name.equals("Keep-Alive", ignoreCase = true) &&
                            !name.equals(HttpHeaders.TransferEncoding, ignoreCase = true) &&
                            !name.equals(HttpHeaders.Upgrade, ignoreCase = true) &&
                            !name.equals("Proxy-Authenticate", ignoreCase = true) &&
                            !name.equals("Proxy-Authorization", ignoreCase = true) &&
                            !name.equals("TE", ignoreCase = true) &&
                            !name.equals("Trailers", ignoreCase = true)) {
                            appendAll(name, values)
                        }
                    }
                }

                // Set body using native Ktor content
                if (bodyBytes != null) {
                    setBody(bodyBytes)
                    contentType(call.request.contentType())
                }

                log("${call.request.httpMethod.value} ${call.request.uri} -> )")
            }.execute { response ->
                // Stream response body using Ktor's low-level OutgoingContent API for true streaming
                // Use WriteChannelContent for raw streaming control
                call.respond(object : io.ktor.http.content.OutgoingContent.WriteChannelContent() {
                    override val status = response.status
                    override val contentType = response.contentType()

                    override val headers = io.ktor.http.HeadersBuilder().apply {
                        response.headers.forEach { name, values ->
                            if (!name.equals(HttpHeaders.TransferEncoding, ignoreCase = true) &&
                                !name.equals(HttpHeaders.Connection, ignoreCase = true) &&
                                !name.equals("Keep-Alive", ignoreCase = true) &&
                                !name.equals(HttpHeaders.Upgrade, ignoreCase = true) &&
                                !name.equals("Proxy-Authenticate", ignoreCase = true) &&
                                !name.equals("Proxy-Authorization", ignoreCase = true) &&
                                !name.equals("TE", ignoreCase = true) &&
                                !name.equals("Trailers", ignoreCase = true)) {
                                appendAll(name, values)
                            }
                        }
                    }.build()

                    override suspend fun writeTo(channel: ByteWriteChannel) {
                        // Get streaming body channel for true streaming without buffering
                        val responseBody = response.bodyAsChannel()

                        // Detect SSE for logging
                        val isSSE = response.contentType()?.match(ContentType.Text.EventStream) == true
                        val isKnownEndpoint = call.request.uri.contains("/api/chat") || call.request.uri.contains("/v1/chat/completions")

                        val buffer = ByteArray(256)
                        var lastLogTime = System.currentTimeMillis()
                        val sseResponses = mutableListOf<String>()

                        while (!responseBody.isClosedForRead) {
                            if (responseBody.availableForRead == 0) {
                                if (!responseBody.awaitContent()) break
                            }

                            val bytesRead = responseBody.readAvailable(buffer, 0, buffer.size)
                            if (bytesRead <= 0) break

                            channel.writeFully(buffer, 0, bytesRead)
                            channel.flush()

                            // Log SSE responses for known endpoints
                            if (isSSE && isKnownEndpoint) {
                                val chunk = buffer.decodeToString(0, bytesRead)
                                extractSSEContent(chunk)?.let { sseResponses.add(it) }

                                // Log progress every 2 seconds
                                val now = System.currentTimeMillis()
                                if (now - lastLogTime > 2000) {
                                    log("  ... streaming ${call.request.uri} (${now - startTime}ms elapsed)")
                                    lastLogTime = now
                                }
                            }
                        }

                        // Log collected SSE response content (up to 300 chars, one line)
                        if (isSSE && isKnownEndpoint && sseResponses.isNotEmpty()) {
                            val combinedResponse = sseResponses.joinToString(" ")
                                .replace("\n", " ").replace("\r", " ")
                                .let { if (it.length > 300) it.take(300) + "..." else it }
                            log("  <- SSE response: $combinedResponse")
                        }
                    }
                })

                // Log response
                val duration = System.currentTimeMillis() - startTime
                val contentType = response.contentType()?.toString() ?: "unknown"
                log("${call.request.httpMethod.value} ${call.request.uri} -> ${response.status.value} ($contentType, ${duration}ms)")
            }

        } catch (e: Exception) {
            val duration = System.currentTimeMillis() - startTime
            log("ERROR: ${call.request.httpMethod.value} ${call.request.uri} -> ${e.message} (${duration}ms)")
            call.respondText("Proxy error: ${e.message}", status = HttpStatusCode.BadGateway)
        }
    }

    /**
     * Proxy WebSocket connections
     * Creates a bidirectional tunnel between client and target server
     * Called from within a webSocket routing block
     */
    private suspend fun DefaultWebSocketServerSession.proxyWebSocketSession(call: ApplicationCall) {
        val targetUrl = Config.TARGET_BASE_URL.replaceFirst("http", "ws") + call.request.uri
        log("WS: ${call.request.uri} -> $targetUrl")

        try {
            // This is the server-side WebSocket session (client connection)
            val serverSession = this

            // Get WebSocket connection parameters
            val url = Url(Config.TARGET_BASE_URL)
            val wsScheme = when (url.protocol.name) {
                "https" -> "wss"
                "http" -> "ws"
                else -> "ws"
            }
            // Connect to target WebSocket using method that supports both ws and wss
            client.webSocket(
                method = HttpMethod.Get,
                host = url.host,
                port = url.port,
                path = call.request.uri,
                request = {
                    this.url.protocol = URLProtocol.createOrDefault(wsScheme)
                }
            ) {
                // This is the client-side WebSocket session (target connection)
                val clientSession = this

                // Create two concurrent jobs for bidirectional proxying
                coroutineScope {
                    // Forward messages from client to target
                    launch {
                        try {
                            for (frame in serverSession.incoming) {
                                clientSession.send(frame.copy())
                            }
                        } catch (e: Exception) {
                            log("WS client->target error: ${e.message}")
                        }
                    }

                    // Forward messages from target to client
                    launch {
                        try {
                            for (frame in clientSession.incoming) {
                                serverSession.send(frame.copy())
                            }
                        } catch (e: Exception) {
                            log("WS target->client error: ${e.message}")
                        }
                    }
                }
            }

            log("WS: ${call.request.uri} closed")
        } catch (e: Exception) {
            log("WS ERROR: ${call.request.uri} -> ${e.message}")
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

    fun stop() {
        server?.stop(1000, 2000)
        client.close()
    }
}
