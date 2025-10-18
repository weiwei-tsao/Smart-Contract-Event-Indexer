const fs = require('fs');
const path = require('path');
const solc = require('solc');

// Read the contract source
const contractPath = path.join(__dirname, 'TestERC20.sol');
const source = fs.readFileSync(contractPath, 'utf8');

// Compile the contract
const input = {
  language: 'Solidity',
  sources: {
    'TestERC20.sol': {
      content: source
    }
  },
  settings: {
    outputSelection: {
      '*': {
        '*': ['*']
      }
    }
  }
};

const output = JSON.parse(solc.compile(JSON.stringify(input)));

if (output.errors) {
  console.error('Compilation errors:', output.errors);
  process.exit(1);
}

// Extract the contract
const contract = output.contracts['TestERC20.sol']['TestERC20'];

// Save ABI and bytecode
const abi = contract.abi;
const bytecode = contract.evm.bytecode.object;

// Create output directory if it doesn't exist
const outputDir = path.join(__dirname, '..', '..', '..', 'services', 'indexer-service', 'testdata');
if (!fs.existsSync(outputDir)) {
  fs.mkdirSync(outputDir, { recursive: true });
}

// Write ABI and bytecode to files
fs.writeFileSync(path.join(outputDir, 'TestERC20.abi.json'), JSON.stringify(abi, null, 2));
fs.writeFileSync(path.join(outputDir, 'TestERC20.bytecode.json'), JSON.stringify(bytecode, null, 2));

console.log('✅ Contract compiled successfully!');
console.log('📁 ABI saved to:', path.join(outputDir, 'TestERC20.abi.json'));
console.log('📁 Bytecode saved to:', path.join(outputDir, 'TestERC20.bytecode.json'));
