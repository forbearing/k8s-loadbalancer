package nginx

/*
upstream #UPSTREAM_NAME# {
}
server {
    listen #LISTEN_PORT# udp;
    proxy_timeout       1m;
    proxy_responses     1;
    proxy_buffer_size   16k;
    proxy_pass          #UPSTREAM_NAME#;
    access_log          /var/log/nginx/#ACCESS_LOG#.log proxy;
}
*/

var TemplateUDP = `
upstream %s {
%s
}
server {
    listen %d udp;
    proxy_timeout       1m;
    proxy_responses     1;
    proxy_buffer_size   16k;
    proxy_pass          %s;
    access_log          /var/log/nginx/%s.log proxy;
}
`
