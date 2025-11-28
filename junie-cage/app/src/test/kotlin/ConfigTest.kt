package org.jonnyzzz.ai.app

import org.junit.jupiter.api.AfterAll
import org.junit.jupiter.api.BeforeAll
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.TestInstance
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertNull
import kotlin.test.assertTrue

/**
 * Tests for Config - model routing configuration
 */
@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class ConfigTest {

    @BeforeAll
    fun setup() {
        // Clear test properties to use production config
        System.clearProperty("test.proxy.port")
        System.clearProperty("test.target.url")
    }

    @AfterAll
    fun teardown() {
        System.clearProperty("test.proxy.port")
        System.clearProperty("test.target.url")
    }

    @Test
    fun `test getBackendForModel returns match for exact model name`() {
        val match = Config.getBackendForModel("llama3.2:latest")

        assertNotNull(match, "Should find matching backend")
        assertEquals("llama3.2:latest", match.inputModel)
        assertEquals("llama3.2:latest", match.targetModel)

        println("✓ Exact model match works")
    }

    @Test
    fun `test gpt-oss model variants match vLLM backend`() {
        // All these should match the vLLM backend pattern
        val variants = listOf(
            "gpt-oss:120b",
            "gpt-oss-120b",
            "hetzner/openai/gpt-oss-120b",
            "some/prefix/gpt-oss:120b"
        )

        for (modelName in variants) {
            val match = Config.getBackendForModel(modelName)
            assertNotNull(match, "Should match vLLM backend for: $modelName")
            assertEquals("gpt-oss:120b", match.targetModel, "Target should be gpt-oss:120b for: $modelName")
            assertTrue(match.backend.useResponsesApi, "Should use Responses API for: $modelName")
            assertTrue(match.backend.backendUrl.contains(":8000"), "Should route to vLLM (port 8000) for: $modelName")
            println("✓ $modelName -> ${match.targetModel} @ ${match.backend.backendUrl}")
        }
    }

    @Test
    fun `test getBackendForModel returns fallback for unknown model`() {
        val match = Config.getBackendForModel("unknown-model-xyz")

        assertNotNull(match, "Should match fallback pattern")
        assertEquals("unknown-model-xyz", match.inputModel)
        assertEquals("qwen3-coder:30b", match.targetModel)  // Fallback target
        assertEquals(".*", match.backend.pattern)

        println("✓ Fallback model routing works")
    }

    @Test
    fun `test getAdvertisedModels returns all advertised models`() {
        val models = Config.getAdvertisedModels()

        assertTrue(models.isNotEmpty(), "Should have advertised models")
        assertTrue(models.contains("llama3.2:latest"), "Should contain llama3.2")
        assertTrue(models.contains("qwen3-coder:30b"), "Should contain qwen3-coder")

        println("✓ getAdvertisedModels returns: $models")
    }

    @Test
    fun `test getExplicitModels returns backends with advertised models`() {
        val backends = Config.getExplicitModels()

        assertTrue(backends.isNotEmpty(), "Should have explicit backends")
        assertTrue(backends.all { it.advertisedModels.isNotEmpty() }, "All explicit backends should have advertised models")

        // Fallback (.*) should NOT be in explicit models
        assertTrue(backends.none { it.pattern == ".*" }, "Fallback should not be in explicit models")

        println("✓ getExplicitModels returns ${backends.size} backends")
    }

    @Test
    fun `test MODEL_BACKENDS order matters for pattern matching`() {
        // First pattern wins, so exact matches should come before wildcards
        val backends = Config.MODEL_BACKENDS
        val wildcardIndex = backends.indexOfFirst { it.pattern == ".*" }

        assertTrue(wildcardIndex == backends.lastIndex, "Wildcard pattern should be last")

        println("✓ Wildcard pattern is correctly last in routing rules")
    }

    @Test
    fun `test test config uses test target url`() {
        System.setProperty("test.target.url", "http://test-server:8000")

        val backends = Config.MODEL_BACKENDS
        assertEquals(1, backends.size, "Test config should have single backend")
        assertEquals("http://test-server:8000", backends.first().backendUrl)
        assertTrue(backends.first().useResponsesApi, "Test backend should use Responses API")

        System.clearProperty("test.target.url")

        println("✓ Test config override works")
    }

    @Test
    fun `test proxy port can be overridden`() {
        System.setProperty("test.proxy.port", "9999")

        assertEquals(9999, Config.PROXY_LISTEN_PORT)

        System.clearProperty("test.proxy.port")

        println("✓ Proxy port override works")
    }

    @Test
    fun `test default proxy port`() {
        System.clearProperty("test.proxy.port")

        assertEquals(1984, Config.PROXY_LISTEN_PORT)

        println("✓ Default proxy port is 1984")
    }
}
