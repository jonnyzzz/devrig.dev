package org.jonnyzzz.ai.app

object Config {
    // Allow overriding for tests via system properties
    val PROXY_LISTEN_PORT: Int
        get() = System.getProperty("test.proxy.port")?.toIntOrNull() ?: 1984

    val TARGET_BASE_URL: String
        get() = System.getProperty("test.target.url") ?: "http://10.212.212.1:11434"

    // Helper functions to construct URLs
    fun buildHttpUrl(path: String): String = "$TARGET_BASE_URL$path"

    val HEALTH_CHECK_URL: String
        get() = "$TARGET_BASE_URL/api/tags"

    const val HEALTH_CHECK_INTERVAL_MS = 100L
}
