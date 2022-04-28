#!/bin/bash
# Shell script for converting a draw.io AWS shapes file into a json file understood by the application.
# The file used for this is pulled directly from GitHub to ensure it always uses the latest
# Usage: ./convert.sh <path to file>
curl --silent https://raw.githubusercontent.com/jgraph/drawio/master/src/main/webapp/js/diagramly/sidebar/Sidebar-AWS4.js | sed -e 's/mxConstants.STYLE_SHAPE/"shape"/g' -e 's/Sidebar\.prototype\.//g' -e '/setCurrentSearchEntryLibrary/d' -e '/})();/d' > convertedfile.js
cat << EOF >> convertedfile.js
	createVertexTemplateEntry = function ( style, width, height, value, title, showLabel, showTitle, tags )
	{
		if (title != null) {
			return '"' + title + '": "' + style + '"'
		}
	};
	createEdgeTemplateEntry = function(style, width, height, value, title, showLabel, tags, allowCellsInserted)
	{
		return "";
	}
	getTagsForStencil = function ( nothing, more, less )
	{
  	return [];
	};
	addPaletteFunctions = function (name, type, something, array)
	{
		myvalues = []
		array.forEach(element => {
			if (element) {
				myvalues.push(element)
			}
		});
		cleanedName = name.replace("aws4", "")
		console.log('"'+cleanedName+'": {'+myvalues.join(",")+"},")
	};

})();
console.log("{")
addAWS4Palette()
console.log("}")
EOF
node convertedfile.js > temp.json
perl -0777 -pe 's/},\n}/}\n}/igs' temp.json | jsonpp -s > aws.json
rm convertedfile.js temp.json
