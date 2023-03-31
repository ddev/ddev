#!/usr/bin/env python

# #ddev-generated: Automatically generated Drupal settings file.
# ddev manages this file and may delete or overwrite the file unless this
# comment is removed.  It is recommended that you leave this file alone.

host = "{{ .DatabaseHost }}"

DATABASES = {
    'default': {
        'ENGINE': '{{ .engine }}',
        'NAME': '{{ .database }}',
        'USER': '{{ .user }}',
        'PASSWORD': '{{ .password }}',
        'HOST': '{{ .host }}',
    },
}

# Override settings here
CSRF_TRUSTED_ORIGINS=["https://*"]
DEBUG = True
EMAIL_BACKEND = "django.core.mail.backends.console.EmailBackend"

ALLOWED_HOSTS = ["*"]


