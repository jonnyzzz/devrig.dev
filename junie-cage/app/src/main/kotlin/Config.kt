package org.jonnyzzz.ai.app

object Config {
    const val PROXY_LISTEN_PORT = 1984
    const val TARGET_HOST = "localhost"
    const val TARGET_PORT = 11434
    const val TARGET_BASE_URL = "http://$TARGET_HOST:$TARGET_PORT"
    const val HEALTH_CHECK_URL = "$TARGET_BASE_URL/api/tags"
    const val HEALTH_CHECK_INTERVAL_MS = 500L
}
