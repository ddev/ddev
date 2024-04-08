#ddev-generated
*.ini files in .ddev/php are added to the project's PHP configuration.

More information is at
https://ddev.readthedocs.io/en/stable/users/extend/customization-extendibility/#custom-php-configuration-phpini

For example, if you rename the provided php-example.ini.example
to php-example.ini its configuration will be added to your PHP configuration.

```
[PHP]
max_execution_time = 240;
```

