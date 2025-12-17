job "moq-relay" {
  datacenters = ["dc1"]
  type        = "service"

  group "relay" {
    count = 1

    network {
      mode = "host"
      port "quic" {
        static = 4433
      }
    }

    task "server" {
      # Use raw_exec to run the binary directly on the host OS.
      # This avoids container networking layers entirely.
      # On Windows, this runs the .exe directly.
      # On Linux/WSL, this runs the ELF binary directly.
      driver = "raw_exec"

      config {
        # Command to execute. 
        # We assume the binary is built into the 'bin' directory (mage nomad:build)
        # For Windows compatibility, we look for the .exe
        command = "bin/qumo-relay.exe"
      }

      # Pass configuration via Environment Variables
      env {
        ADDR = ":${NOMAD_PORT_quic}"
      }

      resources {
        cpu    = 100 # MHz
        memory = 50  # MB
      }

      # Simple health check (TCP check on UDP port won't work, so we skip or use a script)
      # For this dummy UDP example, we'll omit a network health check to keep it simple.
    }
  }
}
