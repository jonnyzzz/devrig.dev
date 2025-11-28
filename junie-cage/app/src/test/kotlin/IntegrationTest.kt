package org.jonnyzzz.ai.app

import io.ktor.client.*
import io.ktor.client.engine.cio.*
import io.ktor.client.plugins.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.utils.io.*
import kotlinx.coroutines.runBlocking
import kotlinx.serialization.json.*
import org.junit.jupiter.api.*
import org.junit.jupiter.api.Assumptions.assumeTrue
import org.junit.jupiter.api.Tag
import kotlin.test.assertEquals
import kotlin.test.assertTrue

/**
 * Integration tests that hit real backends configured in production Config.
 * Run with: ./gradlew integrationTest
 *
 * These tests require the backend servers to be running and accessible.
 * Tests will be SKIPPED if backends are unreachable.
 */
@Tag("integration")
@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class IntegrationTest {

    private lateinit var proxyServer: ProxyServer
    private lateinit var testClient: HttpClient
    private val proxyPort = 19999
    private var backendsReachable = false

    @BeforeAll
    fun setup() {
        System.setProperty("test.proxy.port", proxyPort.toString())
        System.clearProperty("test.target.url")

        testClient = HttpClient(CIO) {
            expectSuccess = false
            install(HttpTimeout) {
                requestTimeoutMillis = 30_000
                connectTimeoutMillis = 5_000
            }
        }

        println("=== Integration Test Setup ===")
        println("Configured model routing rules:")
        Config.MODEL_BACKENDS.forEach {
            println("  - pattern='${it.pattern}' -> target='${it.targetModel}' @ ${it.backendUrl} (responsesApi=${it.useResponsesApi})")
        }

        // Check if backends are reachable before starting proxy
        backendsReachable = checkBackendsReachable()
        println("Backends reachable: $backendsReachable")

        if (backendsReachable) {
            proxyServer = ProxyServer()
            proxyServer.start()
            Thread.sleep(1000)
            println("Proxy running on port: $proxyPort")
        }
        println("==============================")
    }

    private fun checkBackendsReachable(): Boolean = runBlocking {
        for (backend in Config.MODEL_BACKENDS) {
            try {
                val baseUrl = backend.backendUrl.removeSuffix("/v1").removeSuffix("/")
                // Try to connect to the server (any endpoint)
                val checkClient = HttpClient(CIO) {
                    expectSuccess = false
                    install(HttpTimeout) {
                        connectTimeoutMillis = 3_000
                        requestTimeoutMillis = 5_000
                    }
                }
                val response = checkClient.get("$baseUrl/")
                checkClient.close()
                println("✓ Backend $baseUrl is reachable (status: ${response.status})")
                return@runBlocking true
            } catch (e: Exception) {
                println("✗ Backend ${backend.backendUrl} is NOT reachable: ${e.message?.take(100)}")
            }
        }
        false
    }

    @AfterAll
    fun teardown() {
        testClient.close()
        if (backendsReachable) {
            proxyServer.stop()
        }
    }

    @Test
    fun `integration - models endpoint returns configured models`() = runBlocking {
        assumeTrue(backendsReachable, "Skipping: backends not reachable")

        val response = testClient.get("http://localhost:$proxyPort/v1/models")

        assertEquals(HttpStatusCode.OK, response.status)
        val json = Json.parseToJsonElement(response.bodyAsText()).jsonObject
        val models = json["data"]?.jsonArray

        println("Available models: ${models?.map { it.jsonObject["id"]?.jsonPrimitive?.content }}")
        assertTrue(models != null && models.isNotEmpty(), "Should have at least one model")
    }

    @Test
    fun `integration - chat completions non-streaming for each model`() = runBlocking {
        assumeTrue(backendsReachable, "Skipping: backends not reachable")

        for (backend in Config.getExplicitModels()) {
            println("\n--- Testing ${backend.targetModel} (non-streaming) ---")
            println("Backend: ${backend.backendUrl}")
            println("Uses Responses API: ${backend.useResponsesApi}")

            val startTime = System.currentTimeMillis()
            val response = testClient.post("http://localhost:$proxyPort/v1/chat/completions") {
                contentType(ContentType.Application.Json)
                setBody("""{"model":"${backend.targetModel}","messages":[{"role":"user","content":"Say hello in one word"}],"stream":false,"max_tokens":10}""")
            }
            val duration = System.currentTimeMillis() - startTime

            println("Status: ${response.status} (${duration}ms)")

            if (response.status == HttpStatusCode.OK) {
                val json = Json.parseToJsonElement(response.bodyAsText()).jsonObject
                val content = json["choices"]?.jsonArray?.firstOrNull()?.jsonObject
                    ?.get("message")?.jsonObject?.get("content")?.jsonPrimitive?.content
                println("Response: $content")
                assertEquals("chat.completion", json["object"]?.jsonPrimitive?.content)
                assertTrue(content?.isNotEmpty() == true, "Should have response content")
            } else {
                val error = response.bodyAsText()
                println("Error: $error")
                Assertions.fail("Model ${backend.targetModel} returned ${response.status}: $error")
            }
        }
    }

    @Test
    fun `integration - chat completions streaming for each model`() = runBlocking {
        assumeTrue(backendsReachable, "Skipping: backends not reachable")

        for (backend in Config.getExplicitModels()) {
            println("\n--- Testing ${backend.targetModel} (streaming) ---")
            println("Backend: ${backend.backendUrl}")
            println("Uses Responses API: ${backend.useResponsesApi}")

            val startTime = System.currentTimeMillis()
            val chunks = mutableListOf<String>()

            testClient.preparePost("http://localhost:$proxyPort/v1/chat/completions") {
                contentType(ContentType.Application.Json)
                setBody("""{"model":"${backend.targetModel}","messages":[{"role":"user","content":"Say hi"}],"stream":true,"max_tokens":10}""")
            }.execute { response ->
                println("Status: ${response.status}")

                if (response.status == HttpStatusCode.OK) {
                    assertEquals(ContentType.Text.EventStream, response.contentType()?.withoutParameters())

                    val channel = response.bodyAsChannel()
                    while (!channel.isClosedForRead) {
                        val line = channel.readUTF8Line() ?: break
                        if (line.startsWith("data:") && !line.contains("[DONE]")) {
                            val data = line.substring(5).trim()
                            if (data.isNotEmpty()) {
                                try {
                                    val json = Json.parseToJsonElement(data).jsonObject
                                    val content = json["choices"]?.jsonArray?.firstOrNull()?.jsonObject
                                        ?.get("delta")?.jsonObject?.get("content")?.jsonPrimitive?.content
                                    content?.let { chunks.add(it) }
                                } catch (e: Exception) {
                                    println("Parse error: ${e.message}")
                                }
                            }
                        }
                    }
                } else {
                    val error = response.bodyAsText()
                    println("Error: $error")
                    Assertions.fail("Model ${backend.targetModel} streaming returned ${response.status}: $error")
                }
            }

            val duration = System.currentTimeMillis() - startTime
            val fullResponse = chunks.joinToString("")
            println("Chunks received: ${chunks.size}")
            println("Response: $fullResponse")
            println("Duration: ${duration}ms")

            assertTrue(chunks.isNotEmpty(), "Should receive streaming chunks for ${backend.targetModel}")
        }
    }
}
