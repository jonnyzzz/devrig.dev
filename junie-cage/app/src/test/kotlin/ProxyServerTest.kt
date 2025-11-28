package org.jonnyzzz.ai.app

import io.ktor.client.*
import io.ktor.client.engine.cio.*
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
import kotlinx.coroutines.*
import kotlinx.coroutines.test.runTest
import kotlinx.serialization.json.*
import org.junit.jupiter.api.*
import kotlin.test.assertEquals
import kotlin.test.assertTrue
import kotlin.time.Duration.Companion.seconds

/**
 * Tests for ProxyServer - model-based routing proxy
 */
@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class ProxyServerTest {

    private lateinit var mockServer: EmbeddedServer<*, *>
    private lateinit var proxyServer: ProxyServer
    private lateinit var testClient: HttpClient

    private val mockServerPort = 18888
    private val proxyPort = 18889

    @BeforeAll
    fun setup() {
        mockServer = startMockServer()

        System.setProperty("test.proxy.port", proxyPort.toString())
        System.setProperty("test.target.url", "http://localhost:$mockServerPort")

        proxyServer = ProxyServer()
        proxyServer.start()

        Thread.sleep(500)

        testClient = HttpClient(CIO) {
            expectSuccess = false
        }
    }

    @AfterAll
    fun teardown() {
        testClient.close()
        proxyServer.stop()
        mockServer.stop(1000, 2000)
    }

    @Test
    fun `test api tags returns configured models`() = runTest(timeout = 5.seconds) {
        val response = testClient.get("http://localhost:$proxyPort/api/tags")

        assertEquals(HttpStatusCode.OK, response.status)
        assertEquals(ContentType.Application.Json, response.contentType()?.withoutParameters())

        val json = Json.parseToJsonElement(response.bodyAsText()).jsonObject
        val models = json["models"]?.jsonArray

        assertTrue(models != null && models.isNotEmpty(), "Should return at least one model")
        assertEquals("gps-oss:120b", models!!.first().jsonObject["name"]?.jsonPrimitive?.content)

        println("✓ /api/tags returns configured models")
    }

    @Test
    fun `test v1 models returns configured models`() = runTest(timeout = 5.seconds) {
        val response = testClient.get("http://localhost:$proxyPort/v1/models")

        assertEquals(HttpStatusCode.OK, response.status)

        val json = Json.parseToJsonElement(response.bodyAsText()).jsonObject
        assertEquals("list", json["object"]?.jsonPrimitive?.content)

        val models = json["data"]?.jsonArray
        assertTrue(models != null && models.isNotEmpty())
        assertEquals("gps-oss:120b", models!!.first().jsonObject["id"]?.jsonPrimitive?.content)

        println("✓ /v1/models returns configured models")
    }

    @Test
    fun `test models endpoint returns configured models`() = runTest(timeout = 5.seconds) {
        val response = testClient.get("http://localhost:$proxyPort/models")

        assertEquals(HttpStatusCode.OK, response.status)

        val json = Json.parseToJsonElement(response.bodyAsText()).jsonObject
        assertEquals("list", json["object"]?.jsonPrimitive?.content)

        println("✓ /models returns configured models")
    }

    @Test
    fun `test chat completions streaming conversion`() = runTest(timeout = 10.seconds) {
        val receivedChunks = mutableListOf<String>()

        testClient.preparePost("http://localhost:$proxyPort/v1/chat/completions") {
            contentType(ContentType.Application.Json)
            setBody("""{"model":"gps-oss:120b","messages":[{"role":"user","content":"Hello"}],"stream":true}""")
        }.execute { response ->
            assertEquals(HttpStatusCode.OK, response.status)
            assertEquals(ContentType.Text.EventStream, response.contentType()?.withoutParameters())

            val channel = response.bodyAsChannel()
            while (!channel.isClosedForRead) {
                val line = channel.readUTF8Line() ?: break
                if (line.startsWith("data:") && !line.contains("[DONE]")) {
                    receivedChunks.add(line.substring(5).trim())
                }
            }
        }

        assertTrue(receivedChunks.isNotEmpty(), "Should receive streaming chunks")

        val firstChunk = Json.parseToJsonElement(receivedChunks.first()).jsonObject
        assertEquals("chat.completion.chunk", firstChunk["object"]?.jsonPrimitive?.content)

        println("✓ Chat Completions streaming: ${receivedChunks.size} chunks received")
    }

    @Test
    fun `test chat completions non-streaming conversion`() = runTest(timeout = 5.seconds) {
        val response = testClient.post("http://localhost:$proxyPort/v1/chat/completions") {
            contentType(ContentType.Application.Json)
            setBody("""{"model":"gps-oss:120b","messages":[{"role":"user","content":"Test"}],"stream":false}""")
        }

        assertEquals(HttpStatusCode.OK, response.status)

        val json = Json.parseToJsonElement(response.bodyAsText()).jsonObject
        assertEquals("chat.completion", json["object"]?.jsonPrimitive?.content)
        assertTrue(json["choices"]?.jsonArray?.isNotEmpty() == true)

        println("✓ Chat Completions non-streaming works")
    }

    @Test
    fun `test unknown model returns 404`() = runTest(timeout = 5.seconds) {
        val response = testClient.post("http://localhost:$proxyPort/v1/chat/completions") {
            contentType(ContentType.Application.Json)
            setBody("""{"model":"unknown-model","messages":[{"role":"user","content":"Test"}]}""")
        }

        assertEquals(HttpStatusCode.NotFound, response.status)
        val json = Json.parseToJsonElement(response.bodyAsText()).jsonObject
        assertTrue(json["error"]?.jsonObject?.get("message")?.jsonPrimitive?.content?.contains("not found") == true)

        println("✓ Unknown model returns 404")
    }

    @Test
    fun `test unknown endpoint returns 404`() = runTest(timeout = 5.seconds) {
        val response = testClient.post("http://localhost:$proxyPort/v1/unknown") {
            contentType(ContentType.Application.Json)
            setBody("""{"test": true}""")
        }

        assertEquals(HttpStatusCode.NotFound, response.status)

        println("✓ Unknown endpoint returns 404")
    }

    @Test
    fun `test missing model field returns 400`() = runTest(timeout = 5.seconds) {
        val response = testClient.post("http://localhost:$proxyPort/v1/chat/completions") {
            contentType(ContentType.Application.Json)
            setBody("""{"messages":[{"role":"user","content":"Test"}]}""")
        }

        assertEquals(HttpStatusCode.BadRequest, response.status)

        println("✓ Missing model field returns 400")
    }

    private fun startMockServer(): EmbeddedServer<*, *> {
        return embeddedServer(Netty, port = mockServerPort, host = "127.0.0.1") {
            routing {
                // Mock Responses API endpoint (OpenAI-style path)
                post("/v1/responses") {
                    val body = call.receiveText()
                    val json = Json.parseToJsonElement(body).jsonObject
                    val isStream = json["stream"]?.jsonPrimitive?.boolean ?: false

                    if (isStream) {
                        call.respondBytesWriter(contentType = ContentType.Text.EventStream) {
                            repeat(3) { index ->
                                writeStringUtf8("event: response.output_text.delta\n")
                                writeStringUtf8("""data: {"type":"response.output_text.delta","delta":"token$index"}""")
                                writeStringUtf8("\n\n")
                                flush()
                                delay(50)
                            }
                            writeStringUtf8("event: response.completed\n")
                            writeStringUtf8("""data: {"type":"response.completed"}""")
                            writeStringUtf8("\n\n")
                            flush()
                        }
                    } else {
                        call.respondText("""{
                            "id": "resp_123",
                            "output": [{
                                "type": "message",
                                "content": [{"type": "output_text", "text": "Test response"}]
                            }]
                        }""".trimIndent(), ContentType.Application.Json)
                    }
                }
            }
        }.start(wait = false)
    }
}
