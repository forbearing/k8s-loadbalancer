package nginx

/*
upstream #UPSTREAM_NAME# {
}
server {
    listen              #LISTEN_PORT# ssl;
    server_name         "#SERVER_NAME#";

    ssl_certificate     /etc/nginx/ssl/www.example.com.crt;
    ssl_certificate_key /etc/nginx/ssl/www.example.com.key;
    ssl_session_timeout 5m;
    ssl_ciphers         ECDH+AESGCM:DH+AESGCM:ECDH+AES256:DH+AES256:ECDH+AES128:DH+AES:ECDH+3DES:DH+3DES:RSA+AESGCM:RSA+AES:RSA+3DES:!aNULL:!MD5:!DSS;
    ssl_protocols       TLSv1 TLSv1.1 TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    large_client_header_buffers 8 16k;
    client_max_body_size 10G;

    access_log          /var/log/nginx/#ACCESS_LOG#.log;

    location / {
        proxy_http_version 1.1;
        proxy_set_header    Host              $http_host;
        proxy_set_header    X-Real-IP         $remote_addr;
        proxy_set_header    X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header    X-Forwarded-Proto $scheme;
        proxy_ssl_session_reuse off;
        proxy_pass              https://#UPSTREAM_NAME#;
    }
}
*/

var TemplateHTTPS = `
upstream %s {
%s
}
server {
    listen              %d ssl;
    server_name         "%s";

    ssl_certificate     /etc/nginx/ssl/www.example.com.crt;
    ssl_certificate_key /etc/nginx/ssl/www.example.com.key;
    ssl_session_timeout 5m;
    ssl_ciphers         ECDH+AESGCM:DH+AESGCM:ECDH+AES256:DH+AES256:ECDH+AES128:DH+AES:ECDH+3DES:DH+3DES:RSA+AESGCM:RSA+AES:RSA+3DES:!aNULL:!MD5:!DSS;
    ssl_protocols       TLSv1 TLSv1.1 TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    large_client_header_buffers 8 16k;
    client_max_body_size 10G;

    access_log          /var/log/nginx/%s.log;

    location / {
        proxy_http_version 1.1;
        proxy_set_header    Host              $http_host;
        proxy_set_header    X-Real-IP         $remote_addr;
        proxy_set_header    X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header    X-Forwarded-Proto $scheme;
        proxy_ssl_session_reuse off;
        proxy_pass              https://%s;
    }
}
`
