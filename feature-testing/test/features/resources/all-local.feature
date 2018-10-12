Feature: Work-in-progress dumping ground of DDEV local features

   This feature file is home to all ddev local features that have not yet had the Given/when/then steps fully caputred. Once steps have been fully captured, we will pull the scenarios out of this feature file and put them in a more descriptive one

Scenario: DDEV can configure and start Drupal 6, 7 and 8 projects
GIVEN I have a Dru
Scenario: DDEV can configure and start WordPress projects

Scenario: DDEV can configure and start Typo3 projects

Scenario: DDEV can configure and start Backdrop projects
Given I have a Backdrop project checked out
And I navigate to the project directory
When I run ddev config
Then the configuration auto-detects the project as Backdrop

Given I have a Backdrop project
When I run ddev start
Then my project should be up and running and my Backdrop project available

Scenario: DDEV can configure and start PHP projects
Given I have a generic PHP project
When I run ddev start
Then my project should be up and running with PHP installed

Scenario: DDEV can pull and start projects from backups in S3
Given I have a Drupal 7 project checked out
And I have a project backup stored in S3
When I configure my Drupal 7 project with the S3 parameters
And I run ddev pull
Then my project should be running
And my project should contain the data/files from my backup

Scenario: PHP version can be changed in a DDEV project
Given I have a generic PHP project
And I modify the config.yaml to use PHP version 5.6
When I run ddev start
Then my project should be running
And I can confirm it is using PHP version 5.6

Scenario: I can get a list of current DDEV projects
Given I have a set of DDEV projects
When I run ddev list
Then I should see all DDEV projects

Scenario: I can remove a DDEV project
Given I have a DDEV project
And I navigate to the project directory
When I run ddev remove
Then the project is removed

Scenario: I can remove a DDEV project and all associated data
Given I have a Drupal 7 project
And I navigate to the project directory
When I run ddev remove -R
Then the project is removed
And all project data is removed

Scenario: I can remove all DDEV projects from my machine
Given I have a set of DDEV projects
When I run ddev remove -a
Then I should see all DDEV projects have been removed

Scenario: I can start, stop and restart a DDEV project
Given I have a DDEV project
And I navigate to the project directory
When I run ddev start
Then the project is running and available to view

Given I have a DDEV project
And the project is running and available to view
And I navigate to the project directory
When I run ddev stop
Then the project is stopped and no longer available to view

Given I have a DDEV project
And the project is running and available to view
And I navigate to the project directory
When I run ddev restart
Then the project restarted and available to view

Scenario: I can take a database snapshot of a DDEV project

Scenario: I can restore from a database snapshot of a DDEV project

Scenario: I can run a DDEV project with the NginX web server

Scenario: I can run a DDEV project with the Apache web server

Scenario: I can SSH into my project’s web container and issue commands

Scenario: I can check the version of my DDEV Local install
Given I have DDEV Local installed
When I run ddev version
Then the correct ddev version is returned

Scenario: I can use XDebug to step debug my php project
Given I have a generic PHP project
And I set xdebug_enabled to true in my config.yaml file
When I run ddev start
Then I can configure my IDE to listen to debug messages from my PHP project

Scenario: I can use Apache Solr with my project
Given I have a generic PHP project
And I have configured the project to use Apache Solr
#https://ddev.readthedocs.io/en/latest/users/extend/additional-services/#apache-solr
When I run ddev start
Then a Solr admin console for my project is available

Scenario: I can add a memcached instance to my project
#https://ddev.readthedocs.io/en/latest/users/extend/additional-services/#memcached
