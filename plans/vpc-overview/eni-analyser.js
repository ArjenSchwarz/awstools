#!/usr/bin/env node

const { EC2Client, DescribeNetworkInterfacesCommand, DescribeInstancesCommand, DescribeSubnetsCommand, DescribeVpcEndpointsCommand } = require('@aws-sdk/client-ec2');
const { STSClient, GetCallerIdentityCommand } = require('@aws-sdk/client-sts');

// Initialize AWS clients
const ec2Client = new EC2Client({});
const stsClient = new STSClient({});

// Helper functions
function ipToInt(ip) {
    return ip.split('.').reduce((acc, octet) => (acc << 8) + parseInt(octet), 0) >>> 0;
}

function intToIp(int) {
    return [(int >>> 24) & 255, (int >>> 16) & 255, (int >>> 8) & 255, int & 255].join('.');
}

function parseCidr(cidr) {
    const [network, maskBits] = cidr.split('/');
    const mask = parseInt(maskBits);
    const networkInt = ipToInt(network);
    const totalIps = Math.pow(2, 32 - mask);
    const broadcastInt = networkInt + totalIps - 1;

    return {
        networkInt,
        broadcastInt,
        mask,
        totalIps,
        firstUsable: networkInt + 4,  // AWS reserves .0, .1, .2, .3
        lastUsable: broadcastInt - 1   // AWS reserves broadcast
    };
}

function analyzeContiguousBlocks(usedIps, cidr) {
    const { firstUsable, lastUsable } = parseCidr(cidr);

    // Convert and sort used IPs
    const usedInts = usedIps
        .filter(ip => ip)
        .map(ip => ipToInt(ip))
        .sort((a, b) => a - b);

    const freeBlocks = [];
    let prevIp = firstUsable - 1;

    // Find gaps between used IPs
    for (const usedIp of usedInts) {
        if (usedIp > prevIp + 1) {
            const gapStart = prevIp + 1;
            const gapEnd = usedIp - 1;

            if (gapStart <= lastUsable) {
                const actualEnd = Math.min(gapEnd, lastUsable);
                const gapSize = actualEnd - gapStart + 1;

                if (gapSize > 0) {
                    freeBlocks.push({
                        start: gapStart,
                        end: actualEnd,
                        size: gapSize
                    });
                }
            }
        }
        prevIp = usedIp;
    }

    // Check gap after last used IP
    if (prevIp < lastUsable) {
        const gapStart = Math.max(prevIp + 1, firstUsable);
        const gapSize = lastUsable - gapStart + 1;

        if (gapSize > 0) {
            freeBlocks.push({
                start: gapStart,
                end: lastUsable,
                size: gapSize
            });
        }
    }

    return freeBlocks;
}

async function getVpcEndpoints() {
    try {
        const response = await ec2Client.send(new DescribeVpcEndpointsCommand({}));
        return response.VpcEndpoints || [];
    } catch (error) {
        console.error('Error fetching VPC endpoints:', error.message);
        return [];
    }
}

async function checkAwsCredentials() {
    try {
        await stsClient.send(new GetCallerIdentityCommand({}));
        return true;
    } catch (error) {
        console.error('Error: AWS credentials not configured or invalid:', error.message);
        return false;
    }
}

async function getInstanceName(instanceId) {
    if (!instanceId) return 'N/A';

    try {
        const response = await ec2Client.send(new DescribeInstancesCommand({
            InstanceIds: [instanceId]
        }));

        const instance = response.Reservations[0]?.Instances[0];
        const nameTag = instance?.Tags?.find(tag => tag.Key === 'Name');
        return nameTag?.Value || 'N/A';
    } catch {
        return 'N/A';
    }
}

async function getEnis() {
    try {
        const response = await ec2Client.send(new DescribeNetworkInterfacesCommand({}));
        return response.NetworkInterfaces || [];
    } catch (error) {
        console.error('Error fetching ENIs:', error.message);
        return [];
    }
}

async function getSubnets() {
    try {
        const response = await ec2Client.send(new DescribeSubnetsCommand({}));
        return response.Subnets || [];
    } catch (error) {
        console.error('Error fetching subnets:', error.message);
        return [];
    }
}

async function main() {
    // Check AWS credentials
    if (!(await checkAwsCredentials())) {
        process.exit(1);
    }

    console.log('Fetching all ENIs and their details...');
    console.log('='.repeat(50));

    // Get ENI data
    const enis = await getEnis();
    if (enis.length === 0) {
        console.log('No ENIs found or error occurred');
        process.exit(1);
    }

    // Get VPC endpoints to map ENI IDs to endpoint IDs
    const vpcEndpoints = await getVpcEndpoints();
    const eniToEndpointMap = {};

    for (const endpoint of vpcEndpoints) {
        if (endpoint.NetworkInterfaceIds) {
            for (const eniId of endpoint.NetworkInterfaceIds) {
                eniToEndpointMap[eniId] = {
                    endpointId: endpoint.VpcEndpointId,
                    serviceName: endpoint.ServiceName
                };
            }
        }
    }

    // Process ENI data
    const eniData = [];

    for (const eni of enis) {
        const eniId = eni.NetworkInterfaceId;
        const privateIp = eni.PrivateIpAddress;
        const publicIp = eni.Association?.PublicIp || 'N/A';
        const subnetId = eni.SubnetId;
        const status = eni.Status;
        const interfaceType = eni.InterfaceType || 'interface';
        const description = eni.Description || 'N/A';

        // Determine attachment
        const attachment = eni.Attachment;
        const instanceId = attachment?.InstanceId;
        const vpcEndpointInfo = eniToEndpointMap[eniId];

        let attachedTo;
        if (vpcEndpointInfo) {
            // VPC Endpoint attachment
            const serviceName = vpcEndpointInfo.serviceName.split('.').pop(); // Get last part (e.g., 's3', 'ec2')
            attachedTo = `VPC Endpoint: ${vpcEndpointInfo.endpointId} (${serviceName})`;
        } else if (instanceId) {
            const instanceName = await getInstanceName(instanceId);
            if (instanceName !== 'N/A') {
                attachedTo = `Instance: ${instanceId} (${instanceName})`;
            } else {
                attachedTo = `Instance: ${instanceId}`;
            }
        } else if (interfaceType !== 'interface') {
            attachedTo = `Service: ${interfaceType}`;
        } else if (['elb', 'rds', 'lambda', 'vpc'].some(service =>
            description.toLowerCase().includes(service))) {
            attachedTo = `Service: ${description}`;
        } else {
            attachedTo = 'Unattached';
        }

        eniData.push({
            privateIp,
            publicIp,
            attachedTo,
            eniId,
            status,
            subnetId
        });
    }

    // Sort by IP address
    eniData.sort((a, b) => ipToInt(a.privateIp) - ipToInt(b.privateIp));

    // Display ENI table
    console.log(`${'Private IP'.padEnd(15)} ${'Public IP'.padEnd(15)} ${'Attached To'.padEnd(35)} ${'ENI ID'.padEnd(20)} ${'Status'.padEnd(10)}`);
    console.log(`${'-'.repeat(15)} ${'-'.repeat(15)} ${'-'.repeat(35)} ${'-'.repeat(20)} ${'-'.repeat(10)}`);

    for (const eni of eniData) {
        console.log(`${eni.privateIp.padEnd(15)} ${eni.publicIp.padEnd(15)} ${eni.attachedTo.padEnd(35)} ${eni.eniId.padEnd(20)} ${eni.status.padEnd(10)}`);
    }

    console.log('='.repeat(50));
    console.log(`Total ENIs: ${eniData.length}`);
    console.log();

    // Analyze contiguous blocks by subnet
    console.log('Analyzing contiguous free IP blocks by subnet...');
    console.log('='.repeat(60));

    // Group ENIs by subnet
    const subnetEnis = {};
    for (const eni of eniData) {
        if (!subnetEnis[eni.subnetId]) {
            subnetEnis[eni.subnetId] = [];
        }
        subnetEnis[eni.subnetId].push(eni.privateIp);
    }

    // Get subnet information
    const subnets = await getSubnets();

    // Sort subnets by CIDR range (network address)
    subnets.sort((a, b) => {
        const networkA = ipToInt(a.CidrBlock.split('/')[0]);
        const networkB = ipToInt(b.CidrBlock.split('/')[0]);
        return networkA - networkB;
    });

    for (const subnet of subnets) {
        const subnetId = subnet.SubnetId;
        const cidrBlock = subnet.CidrBlock;
        const az = subnet.AvailabilityZone;

        console.log(`\nSubnet: ${subnetId} (${cidrBlock}) - ${az}`);
        console.log('-'.repeat(50));

        // Calculate subnet statistics
        const { totalIps } = parseCidr(cidrBlock);
        const awsReserved = 5; // AWS always reserves 5 IPs
        const totalUsableIps = totalIps - awsReserved;

        const usedIps = subnetEnis[subnetId] || [];
        const usedCount = usedIps.length;
        const freeCount = totalUsableIps - usedCount;

        console.log(`Total subnet IPs: ${totalIps}`);
        console.log(`AWS reserved IPs: ${awsReserved}`);
        console.log(`Total usable IPs: ${totalUsableIps}`);
        console.log(`Used IPs: ${usedCount}`);
        console.log(`Free IPs: ${freeCount}`);

        if (usedIps.length > 0) {
            console.log('Used IP addresses:');
            const sortedUsedIps = usedIps.sort((a, b) => ipToInt(a) - ipToInt(b));

            for (let i = 0; i < Math.min(10, sortedUsedIps.length); i++) {
                console.log(`  ${sortedUsedIps[i]}`);
            }

            if (sortedUsedIps.length > 10) {
                console.log(`  ... and ${sortedUsedIps.length - 10} more`);
            }

            console.log('\nContiguous free IP blocks:');
            const freeBlocks = analyzeContiguousBlocks(usedIps, cidrBlock);

            // Filter and display blocks of 5+ IPs
            const largeBlocks = freeBlocks.filter(block => block.size >= 5);

            if (largeBlocks.length > 0) {
                largeBlocks.forEach((block, i) => {
                    const startIp = intToIp(block.start);
                    const endIp = intToIp(block.end);
                    console.log(`  Block ${i + 1}: ${block.size} IPs (${startIp} - ${endIp})`);
                });
            } else {
                console.log('  No large contiguous blocks (5+ IPs) available');
            }

            const maxBlockSize = Math.max(...freeBlocks.map(block => block.size), 0);
            console.log(`  Largest contiguous block: ${maxBlockSize} IPs`);
        } else {
            console.log('No IPs currently in use - entire subnet available');
            console.log(`Largest contiguous block: ${totalUsableIps} IPs`);
        }
    }
}

// Run the script
main().catch(error => {
    console.error('Script error:', error.message);
    process.exit(1);
});