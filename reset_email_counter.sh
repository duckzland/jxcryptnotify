#!/bin/bash

## Simple method for resetting all jobs email sent counter to 0
sed -Ei 's/"email_sent_count": ([0-9]+).*/"email_sent_count": 0/g' config.json