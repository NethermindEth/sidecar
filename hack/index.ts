import * as fs from 'fs';

console.log(process.argv);

const sidecarPath = process.argv[2];
const blocklakePath = process.argv[3];

type Row = {
	operator: string;
	avs: string;
	snapshot: string;
}

const sidecar = <Row[]>JSON.parse(fs.readFileSync(sidecarPath).toString());
const blocklake = <Row[]>JSON.parse(fs.readFileSync(blocklakePath).toString());

type MappedRows = {
	// avs --> operator --> []snapshot
	[index: string]: {
		[index: string]: string[]
	}
}

const operatorAvsMap: MappedRows = {};

console.log(`Blocklake length: ${blocklake.length}`);
blocklake.forEach(row => {
	if (!operatorAvsMap[row.avs]) {
		operatorAvsMap[row.avs] = {};
	}
	if (!operatorAvsMap[row.avs][row.operator]) {
		operatorAvsMap[row.avs][row.operator] = [];
	}
	operatorAvsMap[row.avs][row.operator].push(row.snapshot);
});


let errors = 0;
console.log(`Sidecar length: ${sidecar.length}`);
sidecar.forEach(row => {
	if (!operatorAvsMap[row.avs] || !operatorAvsMap[row.avs][row.operator]) {
		errors++;
		console.log(`Missing operator ${row.operator} for avs ${row.avs}`);
	} else if(!operatorAvsMap[row.avs][row.operator].includes(row.snapshot)) {
		console.log(`Missing snapshot ${row.snapshot} for operator ${row.operator} and avs ${row.avs}`);
		errors++;
	}
})
console.log(errors);
