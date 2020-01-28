# CircleCI Script Usage

## Tags/Releases build automatically

If you just push a tag, circleci will build that tag and make its artifacts available both on CircleCI and by creating a GitHub release draft containing the artifacts. This is a result of the tag_build *workflow*, which calls the job.
