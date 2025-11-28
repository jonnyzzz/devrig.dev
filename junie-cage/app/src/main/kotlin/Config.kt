package org.jonnyzzz.ai.app

data class ModelBackend(
    val name: String,
    val backendUrl: String,
    val useResponsesApi: Boolean = false
)

object Config {
    val PROXY_LISTEN_PORT: Int
        get() = System.getProperty("test.proxy.port")?.toIntOrNull() ?: 1984

    const val HEALTH_CHECK_INTERVAL_MS = 100L

    val MODEL_BACKENDS: List<ModelBackend>
        get() {
            System.getProperty("test.target.url")?.let {
                return listOf(ModelBackend("gps-oss:120b", it, useResponsesApi = true))
            }
            return listOf(
                ModelBackend("gps-oss:120b", "http://spark-07.labs.intellij.net:8000/v1", useResponsesApi = true),
                // Ollama - uses native Chat Completions API (OpenAI compatible)
                ModelBackend("llama3.2:latest", "http://spark-07.labs.intellij.net:11434", useResponsesApi = false),
                ModelBackend("JetBrains/Mellum-4b-base:latest", "http://spark-07.labs.intellij.net:11434", useResponsesApi = false),
                ModelBackend("qwen3-coder:30b", "http://spark-07.labs.intellij.net:11434", useResponsesApi = false),
            )
        }

    fun getBackendForModel(model: String): ModelBackend? =
        MODEL_BACKENDS.find { it.name == model }
}
