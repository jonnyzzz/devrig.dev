const sharp = require('sharp');
const fs = require('fs');
const path = require('path');

const inputFile = path.join(__dirname, '../static/logo.svg');
const outputFile = path.join(__dirname, '../static/logo.png');

// Convert SVG to PNG with social media optimal dimensions
// LinkedIn recommends 1200x627 pixels for Open Graph images
sharp(inputFile)
  .resize(1200, 1200, {
    fit: 'contain',
    background: { r: 255, g: 255, b: 255, alpha: 1 }
  })
  .png()
  .toFile(outputFile)
  .then(info => {
    console.log('Logo converted successfully!');
    console.log('Output:', outputFile);
    console.log('Dimensions:', info.width, 'x', info.height);
  })
  .catch(err => {
    console.error('Error converting logo:', err);
    process.exit(1);
  });
