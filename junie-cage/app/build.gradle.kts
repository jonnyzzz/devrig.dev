plugins {
    // Kotlin and Compose for Desktop
    kotlin("jvm") version "2.2.21"
    kotlin("plugin.compose") version "2.2.21"
    id("org.jetbrains.compose") version "1.9.3"
    application
}

repositories {
    mavenCentral()
    google()
}

dependencies {
    implementation(compose.desktop.currentOs)
    implementation(compose.material3)
}

kotlin {
    jvmToolchain(21)
}

application {
    // Define the Fully Qualified Name for the application main class
    mainClass = "org.jonnyzzz.ai.app.AppKt"
}
