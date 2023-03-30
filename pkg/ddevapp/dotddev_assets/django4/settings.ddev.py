#!/usr/bin/env python

# {{ $config.Signature }}: Automatically generated Drupal settings file.
# ddev manages this file and may delete or overwrite the file unless this
# comment is removed.  It is recommended that you leave this file alone.

host = "{{ $config.DatabaseHost }}"
port = {{ $config.DatabasePort }}
driver = "{{ $config.DatabaseDriver }}"

DATABASES = {
    'default': {
        'ENGINE': 'django.db.backends.postgresql',
        'NAME': 'db',
        'USER': 'db',
        'PASSWORD': 'db',
        'HOST': 'db',
    },
}

# Override settings here
CSRF_TRUSTED_ORIGINS=["https://*"]
DEBUG = True
EMAIL_BACKEND = "django.core.mail.backends.console.EmailBackend"

# WAGTAILADMIN_BASE_URL required for notification emails
WAGTAILADMIN_BASE_URL = "https://bakerydemo.ddev.site"

ALLOWED_HOSTS = ["*"]

# $databases['default']['default'] = array(
#     'database' => "{{ $config.DatabaseName }}",
# 'username' => "{{ $config.DatabaseUsername }}",
# 'password' => "{{ $config.DatabasePassword }}",
# 'host' => $host,
# 'driver' => $driver,
# 'port' => $port,
# 'prefix' => "{{ $config.DatabasePrefix }}",
# );


