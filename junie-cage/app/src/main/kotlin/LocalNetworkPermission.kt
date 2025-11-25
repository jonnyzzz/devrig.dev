package org.jonnyzzz.ai.app

import kotlinx.coroutines.*

/**
 * Triggers macOS Local Network permission dialog by briefly browsing Bonjour services.
 * Safe to call multiple times; it will only run once per process.
 */
object LocalNetworkPermission {
    @Volatile
    private var started = false

    private var job: Job? = null

    fun requestOnce() {
        if (started) return
        if (!isMac()) return
        started = true

        job = CoroutineScope(Dispatchers.IO + CoroutineName("LocalNetworkPermission")).launch {
            // Using JmDNS via reflection (to avoid hard dependency compilation errors in tooling)
            var jmdnsInstance: Any? = null
            try {
                val jmdnsClass = Class.forName("javax.jmdns.JmDNS")
                val createMethod = jmdnsClass.getMethod("create")
                jmdnsInstance = createMethod.invoke(null)

                // Build a dynamic proxy for ServiceListener
                val listenerInterface = Class.forName("javax.jmdns.ServiceListener")
                val listener = java.lang.reflect.Proxy.newProxyInstance(
                    listenerInterface.classLoader,
                    arrayOf(listenerInterface)
                ) { _, _, _ -> null }

                val addListener = jmdnsClass.getMethod("addServiceListener", String::class.java, listenerInterface)
                val removeListener = jmdnsClass.getMethod("removeServiceListener", String::class.java, listenerInterface)

                val types = listOf("_workstation._tcp.local.", "_http._tcp.local.")
                types.forEach { type -> addListener.invoke(jmdnsInstance, type, listener) }

                // Keep it alive briefly to ensure the OS displays the dialog if needed
                delay(3_000)

                types.forEach { type -> removeListener.invoke(jmdnsInstance, type, listener) }
            } catch (_: Throwable) {
                // Ignore; the worst case is that permission isn't triggered here
            } finally {
                try {
                    jmdnsInstance?.javaClass?.getMethod("close")?.invoke(jmdnsInstance)
                } catch (_: Throwable) {}
            }
        }
    }

    fun stop() {
        job?.cancel()
        job = null
    }

    private fun isMac(): Boolean = System.getProperty("os.name").lowercase().contains("mac")
}
