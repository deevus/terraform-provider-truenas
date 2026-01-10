# Create a custom Docker Compose app
resource "truenas_app" "nginx" {
  name       = "nginx"
  custom_app = true

  compose_config = <<-EOT
    services:
      nginx:
        image: nginx:latest
        ports:
          - "8080:80"
        volumes:
          - config:/etc/nginx/conf.d
          - data:/usr/share/nginx/html
        restart: unless-stopped
    EOT

  storage {
    volume_name      = "config"
    type             = "host_path"
    host_path        = "/mnt/tank/apps/nginx/config"
    auto_permissions = true
  }

  storage {
    volume_name      = "data"
    type             = "host_path"
    host_path        = "/mnt/tank/apps/nginx/html"
    auto_permissions = true
  }

  network {
    port_name   = "web"
    bind_mode   = "published"
    port_number = 8080
  }

  labels = ["nginx", "web"]
}

# Output the app state
output "nginx_state" {
  value = truenas_app.nginx.state
}
