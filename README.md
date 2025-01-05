# KindExport

This is a discord bot that takes substack
articles and sends them via mail to a user.
If configured correctly, it will be automatically sent to the mail
address which belongs to a kindle device and will be available to read.

## Generate jet models from sqlite source
```bash
jet -source=sqlite -dsn="./kindExport.sqlite" -path=./generated
```