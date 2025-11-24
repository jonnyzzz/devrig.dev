package org.jonnyzzz.ai.app

import io.ktor.client.*
import io.ktor.client.engine.cio.CIO as ClientCIO
import io.ktor.client.plugins.*
import io.ktor.client.plugins.websocket.WebSockets as ClientWebSockets
import io.ktor.client.plugins.websocket.webSocket
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.server.application.*
import io.ktor.server.cio.CIO as ServerCIO
import io.ktor.server.engine.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import io.ktor.server.websocket.*
import io.ktor.utils.io.*
import io.ktor.websocket.*
import kotlinx.coroutines.*
import kotlinx.serialization.json.*
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter

/**
 * Reverse Proxy Server with support for HTTP, WebSocket, and SSE
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
        server = embeddedServer(ServerCIO, port = Config.PROXY_LISTEN_PORT, host = "127.0.0.1") {
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
            log("Forwarding to http://${Config.TARGET_HOST}:${Config.TARGET_PORT}")
            log("Supports: HTTP, WebSocket (WS), Server-Sent Events (SSE)")
        }
    }

    /**
     * Proxy HTTP requests including SSE streaming
     * Uses Ktor's native channel-based streaming for efficient forwarding
     */
    private suspend fun proxyHttpRequest(call: ApplicationCall) {
        val targetUrl = "http://${Config.TARGET_HOST}:${Config.TARGET_PORT}${call.request.uri}"
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

            // Use Ktor's native request building with proper header and body handling
            val response: HttpResponse = client.request(targetUrl) {
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
            }

            val duration = System.currentTimeMillis() - startTime

            // Copy response headers using Ktor's native header handling, excluding hop-by-hop headers
            response.headers.forEach { name, values ->
                if (!name.equals(HttpHeaders.TransferEncoding, ignoreCase = true) &&
                    !name.equals(HttpHeaders.Connection, ignoreCase = true) &&
                    !name.equals("Keep-Alive", ignoreCase = true) &&
                    !name.equals(HttpHeaders.Upgrade, ignoreCase = true) &&
                    !name.equals("Proxy-Authenticate", ignoreCase = true) &&
                    !name.equals("Proxy-Authorization", ignoreCase = true) &&
                    !name.equals("TE", ignoreCase = true) &&
                    !name.equals("Trailers", ignoreCase = true)) {
                    values.forEach { value ->
                        call.response.header(name, value)
                    }
                }
            }

            // Stream response body using Ktor's native channel-based response
            // This is critical for SSE (Server-Sent Events) streaming from OpenAI-compatible APIs
            val responseBody = response.bodyAsChannel()
            call.respondBytesWriter(
                contentType = response.contentType(),
                status = response.status
            ) {
                // Use Ktor's native copyTo for efficient streaming
                responseBody.copyTo(this)
            }

            // Log response
            val contentType = response.contentType()?.toString() ?: "unknown"
            log("${call.request.httpMethod.value} ${call.request.uri} -> ${response.status.value} ($contentType, ${duration}ms)")

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
        val targetUrl = "ws://${Config.TARGET_HOST}:${Config.TARGET_PORT}${call.request.uri}"
        log("WS: ${call.request.uri} -> $targetUrl")

        try {
            // This is the server-side WebSocket session (client connection)
            val serverSession = this

            // Connect to target WebSocket
            client.webSocket(
                host = Config.TARGET_HOST,
                port = Config.TARGET_PORT,
                path = call.request.uri
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

    private fun logRequest(call: ApplicationCall, bodyBytes: ByteArray?) {
        val method = call.request.httpMethod.value
        val uri = call.request.uri

        // Check if this is an OpenAI-compatible API request
        if (bodyBytes != null && uri.contains("/v1/chat/completions")) {
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
                        val preview = if (content.length > 100) content.take(100) + "..." else content
                        log("$method $uri$streamIndicator [prompt: $preview]")
                        return
                    }
                } else if (prompt != null) {
                    val preview = if (prompt.length > 100) prompt.take(100) + "..." else prompt
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
