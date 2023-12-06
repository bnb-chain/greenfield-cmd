from web3 import Web3
import requests
import json
import argparse
import subprocess
import random
import string
from urllib.parse import urlparse
import os

def initialize_config_file():
    config_content = """
    rpcAddr = "https://greenfield-chain.bnbchain.org:443"
    chainId = "greenfield_1017-1"
    """

    with open('config.toml', 'w') as file:
        file.write(config_content)

def getJsonAndImage(contract, token_id):
    url = contract.functions.tokenURI(token_id).call()
    print ("get the nft token URI:", url, "token-id", token_id)
    response = requests.get(url)

    if response.status_code == 200:
        data = response.json()
        name = data["name"]
        image_url = data["image"]

        # download nft meta json file
        parsed_url = urlparse(url)
        json_filename = parsed_url.path.split("/")[-1]
        with open(json_filename, 'w') as json_file:
            json.dump(data, json_file)

        # download the image file of nft
        image_response = requests.get(image_url)
        image_prefix = "image"
        if image_response.status_code == 200:
            with open(f"{image_prefix}_{token_id}", 'wb') as image_file:
                image_file.write(image_response.content)

        return name, image_url
    else:
        return None, None

def upload_files_to_bucket(bucket_name, prefix,sp_url):
    bucket_name = bucket_name.lower()
    print ("bucket name:", bucket_name)
    # create bucket
    create_bucket_command = f"./gnfd-cmd -c config.toml -p password.txt bucket create gnfd://{bucket_name}"
    subprocess.run(create_bucket_command, shell=True, check=True)

    # read the image file and upload
    upload_files = []
    for filename in os.listdir('.'):
        if filename.startswith(prefix):
             upload_files.append(filename)

    upload_command = f"./gnfd-cmd -c config.toml -p password.txt object put --visibility public-read {' '.join(upload_files)} gnfd://{bucket_name}"
    subprocess.run(upload_command, shell=True, check=True)


    # read the image file and upload
    for filename in os.listdir('.'):
        if filename.startswith(prefix):
            file_url = f"{sp_url}/view/{bucket_name}/{filename}"
            print("generate image url on greenfield:", file_url)

            if filename.split("_")[-1].isdigit():
                id = int(filename.split("_")[-1])
                json_filename = f"{id}.json"
                if os.path.exists(json_filename):
                    with open(json_filename, 'r') as file:
                       data = json.load(file)
                       data["image"] = file_url
                       with open(json_filename, 'w') as file:
                             json.dump(data, file)

    jsons = [file for file in os.listdir('.') if file.endswith('.json')]
    json_files = ' '.join(jsons)

    upload_json_command = f"./gnfd-cmd -c config.toml -p password.txt object put --visibility public-read {json_files} gnfd://{bucket_name}"
    subprocess.run(upload_json_command, shell=True, check=True)
    for jsonfile in jsons:
       file_url = f"{sp_url}/view/{bucket_name}/{jsonfile}"
       print("generate json url on greenfield:", file_url)

contract_abi = '''[{"inputs":[{"internalType":"string","name":"name","type":"string"},{"internalType":"string","name":"symbol","type":"string"},{"internalType":"contract IERC20","name":"_wad","type":"address"}],"stateMutability":"nonpayable","type":"constructor"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"owner","type":"address"},{"indexed":true,"internalType":"address","name":"approved","type":"address"},{"indexed":true,"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"Approval","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"owner","type":"address"},{"indexed":true,"internalType":"address","name":"operator","type":"address"},{"indexed":false,"internalType":"bool","name":"approved","type":"bool"}],"name":"ApprovalForAll","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"previousOwner","type":"address"},{"indexed":true,"internalType":"address","name":"newOwner","type":"address"}],"name":"OwnershipTransferred","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"uint256","name":"_salePausedTime","type":"uint256"}],"name":"PublicSalePaused","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"uint256","name":"_saleStartTime","type":"uint256"}],"name":"PublicSaleStart","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"uint256","name":"_goldMetaStartingIndex","type":"uint256"},{"indexed":true,"internalType":"uint256","name":"_superMetaStartingIndex","type":"uint256"},{"indexed":true,"internalType":"uint256","name":"_commonMetaStartingIndex","type":"uint256"}],"name":"StartingIndicesSet","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"internalType":"address","name":"from","type":"address"},{"indexed":true,"internalType":"address","name":"to","type":"address"},{"indexed":true,"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"Transfer","type":"event"},{"inputs":[],"name":"MAX_NFT_PURCHASE","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"MWAD_PROVENANCE","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"NUM_COMMON_META","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"NUM_GOLD_META","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"NUM_SUPER_META","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address[]","name":"_addresses","type":"address[]"}],"name":"addAllowListAddress","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"allowListEnabled","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"approve","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"owner","type":"address"}],"name":"balanceOf","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"","type":"address"},{"internalType":"uint256","name":"","type":"uint256"}],"name":"classPurchased","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"commonMetaStartingIndex","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"disableAllowList","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"enableAllowList","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"getApproved","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"getVRFRandomNumber","outputs":[{"internalType":"bytes32","name":"requestId","type":"bytes32"}],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"goldMetaStartingIndex","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"address","name":"operator","type":"address"}],"name":"isApprovedForAll","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"isInAllowList","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"isMinted","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"numMetas","type":"uint256"}],"name":"mint","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"mintPrice","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"_classId","type":"uint256"},{"internalType":"uint256","name":"_amount","type":"uint256"},{"internalType":"address","name":"_receiver","type":"address"}],"name":"mintWithClass","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"mintingStats","outputs":[{"internalType":"uint256","name":"goldClass","type":"uint256"},{"internalType":"uint256","name":"superClass","type":"uint256"},{"internalType":"uint256","name":"commonClass","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"name","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"owner","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"ownerOf","outputs":[{"internalType":"address","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"pausePublicSale","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"planBSetStartingIndices","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"publicSaleActive","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"","type":"address"}],"name":"purchased","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"bytes32","name":"requestId","type":"bytes32"},{"internalType":"uint256","name":"randomness","type":"uint256"}],"name":"rawFulfillRandomness","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"_address","type":"address"}],"name":"removeAllowListAddress","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"renounceOwnership","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"safeTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"},{"internalType":"bytes","name":"_data","type":"bytes"}],"name":"safeTransferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"operator","type":"address"},{"internalType":"bool","name":"approved","type":"bool"}],"name":"setApprovalForAll","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"string","name":"uri","type":"string"}],"name":"setBaseURI","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"setStartingIndicesVRF","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"uint128[]","name":"maxWadHold","type":"uint128[]"},{"internalType":"uint16[]","name":"goldChance","type":"uint16[]"},{"internalType":"uint16[]","name":"superChance","type":"uint16[]"},{"internalType":"uint16[]","name":"commonChance","type":"uint16[]"}],"name":"setupUserChances","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"startPublicSale","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"superMetaStartingIndex","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"bytes4","name":"interfaceId","type":"bytes4"}],"name":"supportsInterface","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"symbol","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"index","type":"uint256"}],"name":"tokenByIndex","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"owner","type":"address"},{"internalType":"uint256","name":"index","type":"uint256"}],"name":"tokenOfOwnerByIndex","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"uint256","name":"_tokenId","type":"uint256"}],"name":"tokenURI","outputs":[{"internalType":"string","name":"","type":"string"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"totalSupply","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[{"internalType":"address","name":"from","type":"address"},{"internalType":"address","name":"to","type":"address"},{"internalType":"uint256","name":"tokenId","type":"uint256"}],"name":"transferFrom","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[{"internalType":"address","name":"newOwner","type":"address"}],"name":"transferOwnership","outputs":[],"stateMutability":"nonpayable","type":"function"},{"inputs":[],"name":"usingChainlinkVRF","outputs":[{"internalType":"bool","name":"","type":"bool"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"vrfRandomResult","outputs":[{"internalType":"uint256","name":"","type":"uint256"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"wad","outputs":[{"internalType":"contract IERC20","name":"","type":"address"}],"stateMutability":"view","type":"function"},{"inputs":[],"name":"withdraw","outputs":[],"stateMutability":"nonpayable","type":"function"}]'''
BucketName = ""

def main():
    parser = argparse.ArgumentParser(description='manual to this script')
    parser.add_argument('--contract', type=str, default = None)
    parser.add_argument('--endpoint', type=str, default= "https://bsc-dataseed3.binance.org/")
    args = parser.parse_args()

    # init greenfield mainnet config
    initialize_config_file()

    w3 = Web3(Web3.HTTPProvider(args.endpoint))

    # Check if connected to the Ethereum node
    if w3.is_connected():
        print("Connected to Ethereum node at https://bsc-dataseed3.binance.org/")
    else:
        print("Failed to connect to Ethereum node at ")
        exit()

    contract_address = Web3.to_checksum_address(args.contract)
    print (contract_address)
    # Create a contract object
    contract = w3.eth.contract(address=contract_address, abi=contract_abi)


    # total_supply2 = contract.functions.inventory().call()
    # get totalSupply
    total_supply = contract.functions.totalSupply().call()
    print ("total supply:", total_supply)
    if total_supply > 5:
        total_supply = 5

    #name, image =  getJsonAndImage(contract, 111421381)
    #print(f"Token ID: {token_id}, Name: {name}, Image: {image}")

    for token_id in range(total_supply):
        # Make a view call to the contract function
        # Print result
        name, image = getJsonAndImage(contract,token_id)
        print(f"Token ID: {token_id}, Name: {name}, Image: {image}")

    random_suffix = ''.join(random.choices(string.ascii_lowercase + string.digits, k=3))
    BucketName = name.split(' ')[0] + "-" + random_suffix
    prefix = "image"
    upload_files_to_bucket(BucketName, prefix, "https://greenfield-sp.4everland.org")

if __name__ == "__main__":
    main()