plugins {
    // Kotlin and Compose for Desktop
    kotlin("jvm") version "2.2.21"
    kotlin("plugin.compose") version "2.2.21"
    kotlin("plugin.serialization") version "2.2.21"
    id("org.jetbrains.compose") version "1.9.3"
}

repositories {
    mavenCentral()
    google()
}

dependencies {
    implementation(compose.desktop.currentOs)
    implementation(compose.material3)
    implementation("io.ktor:ktor-client-okhttp-jvm:3.3.2")

    // Bonjour/mDNS for triggering macOS Local Network permission prompt
    implementation("org.jmdns:jmdns:3.5.10")

    // Ktor Server
    val ktorVersion = "3.3.2"
    implementation("io.ktor:ktor-server-core:$ktorVersion")
    implementation("io.ktor:ktor-server-netty:$ktorVersion")
    implementation("io.ktor:ktor-server-content-negotiation:$ktorVersion")
    implementation("io.ktor:ktor-server-websockets:$ktorVersion")
    implementation("io.ktor:ktor-server-sse:$ktorVersion")

    // Ktor Client
    implementation("io.ktor:ktor-client-core:$ktorVersion")
    implementation("io.ktor:ktor-client-cio:$ktorVersion")
    implementation("io.ktor:ktor-client-okhttp:$ktorVersion")
    implementation("io.ktor:ktor-client-content-negotiation:$ktorVersion")
    implementation("io.ktor:ktor-client-websockets:$ktorVersion")

    // Logging
    implementation("ch.qos.logback:logback-classic:1.5.19")

    // Serialization
    implementation("org.jetbrains.kotlinx:kotlinx-serialization-json:1.7.3")

    // Coroutines for Desktop/Swing
    implementation("org.jetbrains.kotlinx:kotlinx-coroutines-swing:1.10.1")

    // Testing
    testImplementation("io.ktor:ktor-server-test-host:$ktorVersion")
    testImplementation("io.ktor:ktor-client-mock:$ktorVersion")
    testImplementation("org.jetbrains.kotlin:kotlin-test-junit5:2.2.21")
    testImplementation("org.junit.jupiter:junit-jupiter:5.11.4")
    testImplementation("org.jetbrains.kotlinx:kotlinx-coroutines-test:1.10.1")
}

tasks.test {
    useJUnitPlatform()
}

kotlin {
    jvmToolchain(21)
}

// Compose Desktop native packaging (needed to add macOS Info.plist keys)
compose.desktop {
    application {
        mainClass = "org.jonnyzzz.ai.app.AppKt"

        nativeDistributions {
            targetFormats(org.jetbrains.compose.desktop.application.dsl.TargetFormat.Dmg)
            macOS {
                // Use a dedicated Info.plist file to include Local Network keys
                infoPlist {
                    file("src/macos/Info.plist")
                }
                // Note: No special entitlements are required for Bonjour browsing on macOS.
            }
        }
    }
}
