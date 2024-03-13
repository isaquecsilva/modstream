#!/bin/bash

for file in ./public/media/*
do
	echo "Converting $file to mp3 format with 128k bitrate..."
	ffmpeg -i $file -b:a 128k "${file%.*}.mp3"
done