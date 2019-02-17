curl -s --unix-socket /docker/var/run/docker.sock http:/v1.24/images/json | \
jq 'del(.[].Containers, .[].Id, .[].Created, .[].Labels, .[].ParentId, .[].RepoDigests, .[].SharedSize)' -M | \
sed 's/null/["dangling"]/' > /tmp/docker.json 
/usr/bin/python /tmp/main.py
cat /tmp/transformed.json | sed 's/"repotags": "d"/"repotags": "dangling"/' | sed 's/"repotags": "<none>:<none>"/"repotags": "dangling"/' > /tmp/result.json
cat /tmp/result.json