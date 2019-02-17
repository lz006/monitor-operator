import json

with open('/tmp/docker.json') as f:
    data = json.load(f)

for image in data:
  image["RepoTags"] = image["RepoTags"][0]
  image["repotags"] = image.pop("RepoTags")
  image["virtualsize"] = image.pop("VirtualSize")
  image["size"] = image.pop("Size")

with open('/tmp/transformed.json', 'w') as outfile:
    json.dump(data, outfile, indent=4)