package org.jonnyzzz.ai.app

import io.ktor.client.*
import io.ktor.client.engine.cio.CIO as ClientCIO
import io.ktor.client.plugins.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.http.content.*
import io.ktor.server.application.*
import io.ktor.server.cio.CIO as ServerCIO
import io.ktor.server.engine.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.utils.io.*
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.serialization.json.*
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter

class ProxyServer {
    private var server: EmbeddedServer<*, *>? = null
    private val client = HttpClient(ClientCIO) {
        expectSuccess = false

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
            // Use Ktor's native intercept pattern for proxying
            // This intercepts all requests before routing, which is more efficient than routing
            intercept(ApplicationCallPipeline.Call) {
                proxyRequest(call)
            }
        }

        CoroutineScope(Dispatchers.IO).launch {
            server?.start(wait = false)
            log("Proxy server started on http://127.0.0.1:${Config.PROXY_LISTEN_PORT}")
            log("Forwarding to http://${Config.TARGET_HOST}:${Config.TARGET_PORT}")
        }
    }

    private suspend fun proxyRequest(call: ApplicationCall) {
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
            val responseBody = response.bodyAsChannel()
            call.respondBytesWriter(
                contentType = response.contentType(),
                status = response.status
            ) {
                responseBody.copyTo(this)
            }

            // Log response
            log("${call.request.httpMethod.value} ${call.request.uri} -> ${response.status.value} (${duration}ms)")

        } catch (e: Exception) {
            val duration = System.currentTimeMillis() - startTime
            log("ERROR: ${call.request.httpMethod.value} ${call.request.uri} -> ${e.message} (${duration}ms)")
            call.respondText("Proxy error: ${e.message}", status = HttpStatusCode.BadGateway)
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

                if (messages != null) {
                    val lastMessage = messages.lastOrNull()?.jsonObject
                    val content = lastMessage?.get("content")?.jsonPrimitive?.content
                    if (content != null) {
                        val preview = if (content.length > 100) content.take(100) + "..." else content
                        log("$method $uri [prompt: $preview]")
                        return
                    }
                } else if (prompt != null) {
                    val preview = if (prompt.length > 100) prompt.take(100) + "..." else prompt
                    log("$method $uri [prompt: $preview]")
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
