package nginx

/*
upstream #UPSTREAM_NAME# {
}
server {
    listen              #LISTEN_PORT#;
    server_name         "#SERVER_NAME#";

    access_log          /var/log/nginx/#ACCESS_LOG#.log ;

    large_client_header_buffers 8 16k;
    client_max_body_size 10G;

    location / {
        proxy_http_version 1.1;
        proxy_read_timeout 1800;
        proxy_connect_timeout 1800;
        proxy_send_timeout 1800;
        proxy_set_header    Accept-Encoding "";
        proxy_set_header    Host              $http_host;
        proxy_set_header    X-Real-IP         $remote_addr;
        proxy_set_header    X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header    X-Forwarded-By    $server_addr:$server_port;
        proxy_set_header    X-Forwarded-Proto $scheme;
        proxy_pass          http://#UPSTREAM_NAME#;
    }
}
*/

var TemplateHTTP = `
upstream %s {
%s
}
server {
    listen              %d;
    server_name         "%s";

    access_log          /var/log/nginx/%s.log ;

    large_client_header_buffers 8 16k;
    client_max_body_size 10G;

    location / {
        proxy_http_version 1.1;
        proxy_read_timeout 1800;
        proxy_connect_timeout 1800;
        proxy_send_timeout 1800;
        proxy_set_header    Accept-Encoding "";
        proxy_set_header    Host              $http_host;
        proxy_set_header    X-Real-IP         $remote_addr;
        proxy_set_header    X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header    X-Forwarded-By    $server_addr:$server_port;
        proxy_set_header    X-Forwarded-Proto $scheme;
        proxy_pass          http://%s;
    }
}
`
