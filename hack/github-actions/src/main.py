import os
import time
import requests
import json
import logging
logging.basicConfig(level=logging.INFO)

def readInputs():
    githubToken = os.environ.get("GITHUB_TOKEN")
    repositoryOwner = os.environ.get("GITHUB_OWNER")
    repository = os.environ.get("GITHUB_REPO")

    context = os.environ.get("CONTEXT")
    commit_ref = os.environ.get("COMMIT_REF")
    timeout = os.environ.get("TIMEOUT") # milliseconds
    if timeout:
        try:
            timeout = int(timeout)
        except Exception:
            exit('ERROR: Input timeout is not an integer')
    else:
        timeout = 180000

    checkInterval = os.environ.get("CHECK_INTERVAL") # milliseconds
    if checkInterval:
        try:
            checkInterval = int(checkInterval)
        except Exception:
            exit('ERROR: Input checkInterval is not an integer')
    else:
        checkInterval = 60000

    return {
        "context": context,
        "commit_ref": commit_ref,
        "timeout": timeout,
        "checkInterval": checkInterval,
        "githubToken": githubToken,
        "repositoryOwner": repositoryOwner,
        "repository": repository
    }

def printInputs(inputs):
    print('****Using the following configurations:****')
    print('Context : {}'.format(inputs['context']))
    print('Commit REF : {}'.format(inputs['commit_ref']))
    print('Timeout : {}'.format(inputs['timeout']))
    print('Check Interval : {}'.format(inputs['checkInterval']))
    print('Owner : {}'.format(inputs['repositoryOwner']))
    print('Repository : {}'.format(inputs['repository']))

def fetchCommitStatuses(owner, repo, sha, githubToken):
    url = "https://api.github.com/repos/{}/{}/commits/{}/status".format(owner, repo, sha)
    reqHeaders = {
        'Accept': 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28',
        'Authorization' : 'Bearer {}'.format(githubToken)
    }

    logging.info('Fetching commit status from {}'.format(url))
    response = requests.get(url, headers=reqHeaders)
    if response.status_code != 200:
        raise Exception('API call failed. Status code: {}, {}'.format(response.status_code, response.text))
    return response.json() 
    
def filterCommitStatusByContext(context, statuses):
    for status in statuses:
        if context == status['context']:
            return status
    return None

def setActionOutput(name, value):
    with open(os.environ['GITHUB_OUTPUT'], 'a') as fh:
        print(f'{name}={value}', file=fh)

def check_commit_status_for_success(inputs):
    result = {
        "concluded": False,
        "exitCode": 1,
        "commitStatus": {},
    }

    # Fetch commit statuses from GitHub.
    commitStatus = fetchCommitStatuses(inputs['repositoryOwner'], inputs['repository'], inputs['commit_ref'], inputs['githubToken'])

    # Filter the required status by context.
    status = filterCommitStatusByContext(inputs['context'], commitStatus['statuses'])

    # Check if status has a conclusive state.
    if status == None:
        logging.info('Status not found!')
    elif status['state'] == 'pending':
        logging.info('Status state: {}'.format(status['state']))
    elif status['state'] == 'failure':
        result["concluded"] = True
        result["exitCode"] = 1
    elif status['state'] == 'success':
        result["concluded"] = True
        result["exitCode"] = 0
    else:
        logging.info('Unknown status.state: {}'.format(status['state']))
        result["concluded"] = True
        result["exitCode"] = 1

    if status != None:
        result["commitStatus"] = status
    return result

def main():
    inputs = readInputs()
    printInputs(inputs)
    
    startTime = time.time() # seconds
    while True:
        if (time.time() - startTime)*1000 > inputs['timeout']:
            setActionOutput('state', 'timeout')
            logging.info('Action timed out.')
            exit(1)
        
        result = check_commit_status_for_success(inputs)
        if result["concluded"]:
            jsonStr = json.dumps(result["commitStatus"])
            setActionOutput('state', result["commitStatus"]['state'])
            setActionOutput('json', jsonStr)
            print(result["commitStatus"])
            exit(result["exitCode"])

        # Sleep for `checkInterval`
        time.sleep(inputs['checkInterval']/1000) # convert time to seconds.


if __name__ == "__main__":
    main()
