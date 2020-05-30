#!/usr/bin/env sh
for a in $(cat .env); do
   eval "export $a"
done

