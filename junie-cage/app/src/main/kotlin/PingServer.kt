package org.jonnyzzz.ai.app

import kotlinx.coroutines.*
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter

/**
 * Monitors network interface changes and notifies when an adapter becomes available.
 * Uses macOS `ifconfig` command to check interface media status.
 * Detects hardware presence (media != none), not just link status.
 * Reacts within ~50ms of an interface becoming available.
 *
 * Triggers on EVERY interface becoming available, including reconnection of the same adapter.
 */
class PingServer(
    private val onServerDetected: () -> Unit
) {
    private var job: Job? = null
    private val timeFormatter = DateTimeFormatter.ofPattern("HH:mm:ss.SSS")

    // Track known AVAILABLE network interfaces (those with media != none)
    @Volatile
    private var knownAvailableInterfaces: Set<String> = emptySet()

    // Counter for periodic full logging
    private var pollCount = 0L

    fun start() {
        job = CoroutineScope(Dispatchers.IO + CoroutineName("NetworkMonitor")).launch {
            log("=== Network monitor starting (tracking media availability) ===")

            // Initialize with current available interfaces (don't trigger on startup)
            knownAvailableInterfaces = getAvailableInterfaces()
            log("Initial available interfaces: $knownAvailableInterfaces")

            log("=== Starting polling loop (every 50ms) ===")

            // Poll for network interface changes every 50ms
            while (isActive) {
                checkNetworkInterfaces()
                delay(50) // 50ms polling for fast reaction
            }
        }
    }

    private suspend fun checkNetworkInterfaces() {
        pollCount++

        val currentAvailableInterfaces = getAvailableInterfaces()

        // Log full state every 2 seconds (40 polls) for debugging
        if (pollCount % 40 == 0L) {
            log("POLL #$pollCount: available=$currentAvailableInterfaces, known=$knownAvailableInterfaces")
        }

        // Detect interfaces that became unavailable - remove from known set
        val unavailableInterfaces = knownAvailableInterfaces - currentAvailableInterfaces
        if (unavailableInterfaces.isNotEmpty()) {
            log("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
            log(">>> INTERFACE(S) UNAVAILABLE: $unavailableInterfaces")
            log("    available now=$currentAvailableInterfaces")
            log("    known was=$knownAvailableInterfaces")
            // Update known set to remove unavailable interfaces
            knownAvailableInterfaces = knownAvailableInterfaces - unavailableInterfaces
            log("    known now=$knownAvailableInterfaces")
            log("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
        }

        // Detect interfaces that became available
        val newlyAvailableInterfaces = currentAvailableInterfaces - knownAvailableInterfaces
        if (newlyAvailableInterfaces.isNotEmpty()) {
            log("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
            log(">>> INTERFACE(S) AVAILABLE: $newlyAvailableInterfaces")
            log("    available now=$currentAvailableInterfaces")
            log("    known was=$knownAvailableInterfaces")
            // Update known set with newly available interfaces
            knownAvailableInterfaces = knownAvailableInterfaces + newlyAvailableInterfaces
            log("    known now=$knownAvailableInterfaces")

            log(">>> TRIGGERING NOTIFICATION NOW")
            // Trigger callback on main thread
            withContext(Dispatchers.Main) {
                onServerDetected()
            }
            log(">>> NOTIFICATION TRIGGERED")
            log("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
        }
    }

    /**
     * Get network interfaces that have hardware available.
     * An interface is "available" if its media is NOT "none" and NOT "<unknown type>".
     * This detects when a physical adapter is connected, regardless of cable/link status.
     */
    private fun getAvailableInterfaces(): Set<String> {
        return try {
            val process = ProcessBuilder("ifconfig")
                .redirectErrorStream(true)
                .start()

            val output = process.inputStream.bufferedReader().readText()
            val exitCode = process.waitFor()

            if (exitCode != 0) {
                log("WARNING: ifconfig exited with code $exitCode")
                return emptySet()
            }

            // Parse ifconfig output to find interfaces with media available
            val availableInterfaces = mutableSetOf<String>()
            var currentInterface: String? = null
            var hasMedia = false

            for (line in output.lines()) {
                // Interface line starts with interface name (no leading whitespace)
                if (line.isNotEmpty() && !line[0].isWhitespace() && line.contains(":")) {
                    // Save previous interface if it had media
                    if (currentInterface != null && hasMedia) {
                        availableInterfaces.add(currentInterface)
                    }
                    currentInterface = line.substringBefore(":").trim()
                    hasMedia = false
                }
                // Check for media line - available if NOT "none" and NOT "<unknown type>"
                if (line.trimStart().startsWith("media:") && currentInterface != null) {
                    val mediaValue = line.substringAfter("media:").trim().lowercase()
                    hasMedia = !mediaValue.startsWith("none") &&
                               !mediaValue.startsWith("<unknown type>") &&
                               mediaValue.isNotEmpty()
                }
            }
            // Don't forget last interface
            if (currentInterface != null && hasMedia) {
                availableInterfaces.add(currentInterface)
            }

            availableInterfaces
        } catch (e: Exception) {
            log("ERROR running ifconfig: ${e::class.simpleName}: ${e.message}")
            emptySet()
        }
    }

    private fun log(message: String) {
        val timestamp = LocalDateTime.now().format(timeFormatter)
        println("[$timestamp] [NETWORK] $message")
    }

    fun stop() {
        job?.cancel()
        log("Network monitor stopped")
    }
}
