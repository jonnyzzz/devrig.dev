package org.jonnyzzz.ai.app

import io.ktor.client.*
import io.ktor.client.engine.cio.CIO as ClientCIO
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.server.application.*
import io.ktor.server.cio.CIO as ServerCIO
import io.ktor.server.engine.*
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

class ProxyServer {
    private var server: EmbeddedServer<*, *>? = null
    private val client = HttpClient(ClientCIO) {
        expectSuccess = false
        engine {
            requestTimeout = 600_000 // 10 minutes for slow Ollama responses
        }
    }
    private val timeFormatter = DateTimeFormatter.ofPattern("HH:mm:ss.SSS")

    fun start() {
        server = embeddedServer(ServerCIO, port = Config.PROXY_LISTEN_PORT, host = "127.0.0.1") {
            routing {
                route("{...}") {
                    handle {
                        proxyRequest(call)
                    }
                }
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
            // Read body once for logging and proxying
            val bodyBytes = if (call.request.httpMethod in listOf(HttpMethod.Post, HttpMethod.Put, HttpMethod.Patch)) {
                call.receiveChannel().toByteArray()
            } else {
                null
            }

            // Log request with prompt preview for OpenAI API
            logRequest(call, bodyBytes)

            val response: HttpResponse = client.request(targetUrl) {
                method = call.request.httpMethod

                // Copy headers
                call.request.headers.forEach { name, values ->
                    if (!name.equals("Host", ignoreCase = true)) {
                        values.forEach { value ->
                            header(name, value)
                        }
                    }
                }

                // Copy body if present
                if (bodyBytes != null) {
                    setBody(bodyBytes)
                }
            }

            val duration = System.currentTimeMillis() - startTime

            // Copy response status
            call.response.status(response.status)

            // Copy response headers
            response.headers.forEach { name, values ->
                if (!name.equals("Transfer-Encoding", ignoreCase = true)) {
                    values.forEach { value ->
                        call.response.header(name, value)
                    }
                }
            }

            // Copy response body
            val responseBytes = response.readRawBytes()
            call.respondBytes(responseBytes)

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
