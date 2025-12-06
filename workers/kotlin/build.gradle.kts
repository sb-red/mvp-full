plugins {
    kotlin("jvm") version "1.9.20"
    application
}

group = "com.softgate"
version = "1.0.0"

repositories {
    mavenCentral()
}

dependencies {
    implementation("redis.clients:jedis:5.1.0")
    implementation("com.google.code.gson:gson:2.10.1")
    implementation(kotlin("stdlib"))
}

application {
    mainClass.set("com.softgate.WorkerKt")
}

tasks.jar {
    archiveFileName.set("kotlin-${version}.jar")
    manifest {
        attributes["Main-Class"] = "com.softgate.WorkerKt"
    }
    duplicatesStrategy = DuplicatesStrategy.EXCLUDE
    from(configurations.runtimeClasspath.get().map { if (it.isDirectory) it else zipTree(it) })
}
