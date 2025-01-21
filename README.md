# Blog aggregator

## This is a readme file on github

There is supposed to be something here since it's a readme so let's roll (totally didn't forget all markdown!)

## Installation of required frameworks

### Go

To install golang framework run:

>sudo apt install go

### PostgreSQL

To install postgresql database run:

>sudo apt install postgresql postgresql-contrib

## Installation of gator

Just run:

>go install github.com/smallsonxdd/blog_gator@latest

## Setting up config JSON

This is going to sound silly, but... -

In your home directory, create a JSON file called:

>.gatorconfig.json

This config file holds necessary information about feed. Without it, the program will not run regardless of your go/postgres/gator installations.

