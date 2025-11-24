package org.jonnyzzz.ai.app

import io.ktor.client.*
import io.ktor.client.engine.cio.CIO
import io.ktor.client.request.*
import io.ktor.client.statement.*
import kotlinx.coroutines.*
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter

class PingServer(
    private val healthCheckUrl: String = Config.HEALTH_CHECK_URL,
    private val checkIntervalMs: Long = Config.HEALTH_CHECK_INTERVAL_MS,
    private val onServerDetected: () -> Unit
) {
    private val client = HttpClient(CIO) {
        expectSuccess = false
        engine {
            requestTimeout = Config.HEALTH_CHECK_INTERVAL_MS/2
        }
    }

    private var job: Job? = null
    private var wasOnline = true
    private val timeFormatter = DateTimeFormatter.ofPattern("HH:mm:ss.SSS")

    fun start() {
        job = CoroutineScope(Dispatchers.IO + CoroutineName("Ping")).launch {
            log("Health check started for $healthCheckUrl")
            while (isActive) {
                checkHealth()
                delay(checkIntervalMs)
            }
        }
    }

    private suspend fun checkHealth() {
        try {
            val response: HttpResponse = client.get(healthCheckUrl)

            if (response.status.value in 200..299) {
                if (!wasOnline) {
                    log("Target server DETECTED at $healthCheckUrl")
                    wasOnline = true
                    // Notify that server is back online
                    withContext(Dispatchers.Main) {
                        onServerDetected()
                    }
                }
            } else {
                handleOffline("returned status ${response.status.value}")
            }
        } catch (e: Exception) {
            handleOffline(e.message ?: "unknown error")
        }
    }

    private fun handleOffline(reason: String) {
        if (wasOnline) {
            log("Target server LOST at $healthCheckUrl: $reason")
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
