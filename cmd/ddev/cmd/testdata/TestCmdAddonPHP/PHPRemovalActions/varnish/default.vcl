# Simple default VCL.
#ddev-generated
# For a more advanced example see https://github.com/mattiasgeniar/varnish-6.0-configuration-templates

vcl 4.1;
import std;

backend default {
  .host = "web";
  .port = "80";
}


sub vcl_recv {
  if (std.port(server.ip) == 8025) {
    return (synth(750));
  }
}

sub vcl_synth {
  if (resp.status == 750) {
    set resp.status = 301;
    set resp.http.location = req.http.X-Forwarded-Proto + "://novarnish." + req.http.Host + req.url;
    set resp.reason = "Moved";
    return (deliver);
  }
}
