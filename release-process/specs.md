I need to create a script in the @folder:scripts folder, 
which should be able to do the following.


- fetch the `latest.json` from the latest release of devrig from  https://github.com/jonnyzzz/devrig.dev/releases
- donwload all artifacts for the release locally and maintain the map <url> to <local path>
- ugnore donwload urls in the latest.json, just only use the filenames
- validate GitHub's hashes to the local files
- validate sha512 hashes to the local files against the latest.json
- move crypto folder to this folder
- generate the new latest.json with correct download URLs
- use ssh-sign to create the new latest.json.sign file
- upload the new latest.json and latest.json.sign to the website/static/downloads folder


simplify the logic of @file:sync-release.sh, make it work. the lastest.json does not have url key anymore, and we need to add the url key based on github metadata.

My bash does not have declare -A, so we use files to store that data instead

Change the login to download all atrifacts from GitHub first, put respective JSON from GitHub release to a <file>.github on the disk, we use that next.

Extract shasum and validate signature

Extract the browser_download_url and create <file>.url for it.


Once all is done, let's read the `latest.json` file from the disk.

For each entry validate the sha512 of the binary (there must be `.sha512` file on the disk). Add the url from the <file>.url

Write all changes to the latest.final.json.

Use the @file:ssh-sign.sh to sign that file and name it latest.json.sig

