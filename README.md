# cmms

Frontend ---> templates → main file → index.html

Styles used in all HTML files → style/filename.css



# Asset

_id (ObjectId)

label (String)

effective_date (Date)

type (String)




# Consumable

_id (ObjectId)

label (String)

asset_id (ObjectId → Asset._id)

schedule (Array)

id (ObjectId)

type (String)

days (Number)

services (Array of ObjectId → Service._id)

conservation (Array of ObjectId → Conservation._id)

notes (String)



# Service

_id (ObjectId)

label (String)

notes (String)



# Conservation

_id (ObjectId)

label (String)

notes (String)
