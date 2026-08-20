const fs = require('fs');
const j = JSON.parse(fs.readFileSync('docs/apifox/MeteorX-backend.openapi.json', 'utf8'));
console.log('openapi paths:');
for (const p of Object.keys(j.paths).filter(p => /announcement|cancel-request|dashboard/i.test(p))) {
  console.log(p, '=>', Object.keys(j.paths[p]).join(','));
}
