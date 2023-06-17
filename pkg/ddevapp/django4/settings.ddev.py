#!/usr/bin/env python

# #ddev-generated: Automatically generated Drupal settings file.
# ddev manages this file and may delete or overwrite the file unless this
# comment is removed.  It is recommended that you leave this file alone.

DATABASES = {
    'default': {
        'ENGINE': '{{ .engine }}',
        'NAME': '{{ .database }}',
        'USER': '{{ .user }}',
        'PASSWORD': '{{ .password }}',
        'HOST': '{{ .host }}',
    },
}

SECURE_PROXY_SSL_HEADER = ('HTTP_X_FORWARDED_PROTO', 'https')
CSRF_TRUSTED_ORIGINS=["https://*", "http://*"]
DEBUG = True

# Mailhog setup
EMAIL_BACKEND = 'django.core.mail.backends.smtp.EmailBackend'
EMAIL_HOST = '127.0.0.1'
EMAIL_PORT = '1025'

ALLOWED_HOSTS = ["*"]


