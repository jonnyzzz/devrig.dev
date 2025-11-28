package org.jonnyzzz.ai.app

import io.ktor.client.*
import io.ktor.client.engine.okhttp.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import kotlinx.coroutines.*
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter
import kotlin.time.Duration.Companion.milliseconds

class PingServer(
    private val onServerDetected: () -> Unit
) {
    private val client = HttpClient(OkHttp) {
        engine {
            config {
                followRedirects(true)
                readTimeout(50.milliseconds)
                connectTimeout(50.milliseconds)
                writeTimeout(50.milliseconds)
                callTimeout(50.milliseconds)
            }
        }
        expectSuccess = false
    }

    private var job: Job? = null
    private var wasOnline = true
    private val timeFormatter = DateTimeFormatter.ofPattern("HH:mm:ss.SSS")

    fun start() {
        job = CoroutineScope(Dispatchers.IO + CoroutineName("Ping")).launch {
            val backends = Config.MODEL_BACKENDS.map { it.backendUrl }.distinct()
            log("Health check started for backends: $backends")
            while (isActive) {
                checkHealth(backends)
                delay(Config.HEALTH_CHECK_INTERVAL_MS)
            }
        }
    }

    private suspend fun checkHealth(backends: List<String>) {
        var anyOnline = false
        for (backend in backends) {
            try {
                val response: HttpResponse = client.get("$backend/models")
                if (response.status.value in 200..299) {
                    anyOnline = true
                }
            } catch (_: Exception) {
                // Backend offline
            }
        }

        if (anyOnline && !wasOnline) {
            log("Backend(s) DETECTED")
            wasOnline = true
            withContext(Dispatchers.Main) {
                onServerDetected()
            }
        } else if (!anyOnline && wasOnline) {
            log("All backends LOST")
            wasOnline = false
        }
    }

    private fun log(message: String) {
        val timestamp = LocalDateTime.now().format(timeFormatter)
        println("[$timestamp] [PING] $message")
    }

    fun stop() {
        job?.cancel()
        client.close()
    }
}
