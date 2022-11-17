# Tweety-Collector

## Version

Current stable version 1.10.0 as of July 28th 2021

## Description

Tweety-Collector is a microservice application as part of Tweety application.
Tweety-Collector scrapes user data using [Twitter API v1.1](https://developer.twitter.com/en/docs/twitter-api). It passes scraped data to other microservices who process them further.
Specifically, Tweety-Collector scrapes user metadata and sends it to a microservice application that stores data in database.
It also scrapes list of users friends ids and sends the list to a microservice which gets tweets for given ids.
Moreover, for every scraped user data, Tweety-Collector scrapes its location data (if user has its location field set to public) using [REST Countries API v2](https://restcountries.eu/) and sends scraped data to the microservice which stores location data to database.

## History

Collector was created by Luka Brecic with a little help from his friends (quote on Joe Cocker), [Ana Leventic](https://gitlab.com/leapbit-practice/tweety-dbsaver) and [Josip Srzic](https://gitlab.com/leapbit-practice/tweety-counter) who also participated in creation of other microservices for Tweety application. (Check 'em out :-) !)
Whole [Tweety project](https://gitlab.com/leapbit-practice) started as an internship task at Leapbit corp. led by our kind and wise mentors Luka Brletic and Anton Ruzman from whom I have learned a lot, and for that, I am endlessly thankful.

## Tweety-Collector docs

see documentation [here](docs.md).