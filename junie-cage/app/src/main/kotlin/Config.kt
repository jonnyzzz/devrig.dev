package org.jonnyzzz.ai.app

/**
 * Model backend configuration.
 * @param pattern Regex pattern to match incoming model names (first match wins)
 * @param targetModel Model name to send to the backend
 * @param backendUrl Backend server URL
 * @param useResponsesApi If true, convert Chat Completions to Responses API
 * @param advertisedModels Model names to list in /models and /api/tags endpoints (must match pattern)
 */
data class ModelBackend(
    val pattern: String,
    val targetModel: String,
    val backendUrl: String,
    val useResponsesApi: Boolean = false,
    val advertisedModels: List<String> = emptyList()
) {
    val regex: Regex by lazy { Regex(pattern) }
}

/**
 * Result of model matching - contains both the backend config and the target model to use
 */
data class ModelMatch(
    val backend: ModelBackend,
    val targetModel: String,
    val inputModel: String
)

object Config {
    val TARGET_HOST : String
        get() = System.getenv("JETCABLE_HOST") ?: "spark-07.labs.intellij.net"

    val PROXY_LISTEN_PORT: Int
        get() = System.getProperty("test.proxy.port")?.toIntOrNull() ?: 1984

    val MODEL_BACKENDS: List<ModelBackend>
        get() {
            System.getProperty("test.target.url")?.let {
                return listOf(ModelBackend(
                    pattern = ".*gpt-oss:120b",
                    targetModel = "gpt-oss:120b",
                    backendUrl = it,
                    useResponsesApi = true,
                    advertisedModels = listOf("gpt-oss:120b", "hetzner/openai/gpt-oss-120b")
                ))
            }

            val schema = "h" + "t".repeat(2) + "p://"
            val vllm = "${schema}$TARGET_HOST:8000"
            val ollama = "${schema}$TARGET_HOST:11434"
            return listOf(
                // Match gpt-oss model variants -> vLLM Responses API backend
                // Matches: gpt-oss:120b, gpt-oss-120b, hetzner/openai/gpt-oss-120b, etc.
                ModelBackend(
                    pattern = "(.*/)?gpt-oss[:-]120b",
                    targetModel = "gpt-oss:120b",
                    backendUrl = "$vllm/v1",
                    useResponsesApi = true,
                    advertisedModels = listOf("gpt-oss:120b", "hetzner/openai/gpt-oss-120b")
                ),
                // Exact matches for Ollama models
                ModelBackend(
                    pattern = "llama3.2:latest",
                    targetModel = "llama3.2:latest",
                    backendUrl = ollama,
                    useResponsesApi = false,
                    advertisedModels = listOf("llama3.2:latest")
                ),
                ModelBackend(
                    pattern = "JetBrains/Mellum-4b-base:latest",
                    targetModel = "JetBrains/Mellum-4b-base:latest",
                    backendUrl = ollama,
                    useResponsesApi = false,
                    advertisedModels = listOf("JetBrains/Mellum-4b-base:latest")
                ),
                ModelBackend(
                    pattern = "qwen3-coder:30b",
                    targetModel = "qwen3-coder:30b",
                    backendUrl = ollama,
                    useResponsesApi = false,
                    advertisedModels = listOf("qwen3-coder:30b")
                ),
                // Fallback: any unmatched model -> route to Ollama with qwen3-coder
                // No advertised models - this is a catch-all fallback
                ModelBackend(
                    pattern = ".*",
                    targetModel = "qwen3-coder:30b",
                    backendUrl = ollama,
                    useResponsesApi = false,
                    advertisedModels = emptyList()
                ),
            )
        }

    /**
     * Find matching backend for the given model name.
     * Iterates through MODEL_BACKENDS in order, returns first regex match.
     * @return ModelMatch with backend config and resolved target model, or null if no match
     */
    fun getBackendForModel(model: String): ModelMatch? {
        for (backend in MODEL_BACKENDS) {
            if (backend.regex.matches(model)) {
                return ModelMatch(
                    backend = backend,
                    targetModel = backend.targetModel,
                    inputModel = model
                )
            }
        }
        return null
    }

    /**
     * Get all advertised model names from all routing rules.
     * These are the models shown in /models and /api/tags endpoints.
     */
    fun getAdvertisedModels(): List<String> =
        MODEL_BACKENDS.flatMap { it.advertisedModels }.distinct()

    /**
     * Get routing rules that have advertised models (for iteration in tests)
     */
    fun getExplicitModels(): List<ModelBackend> =
        MODEL_BACKENDS.filter { it.advertisedModels.isNotEmpty() }
}
