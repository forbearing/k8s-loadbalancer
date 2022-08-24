package template

/*
upstream #UPSTREAM_NAME# {
}
server {
    listen #LISTEN_PORT#;
    proxy_timeout       1m;
    proxy_responses     1;
    proxy_buffer_size   16k;
    proxy_pass          #UPSTREAM_NAME#;
    access_log          /var/log/nginx/#ACCESS_LOG#.log proxy;
}
*/

var TemplateTCP = `
upstream %s {
%s
}
server {
    listen %d;
    proxy_timeout       1m;
    proxy_responses     1;
    proxy_buffer_size   16k;
    proxy_pass          %s;
    access_log          /var/log/nginx/%s.log proxy;
}
`
