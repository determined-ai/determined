package com.determined.codegen;

import io.swagger.codegen.SupportingFile;
import io.swagger.codegen.languages.TypeScriptFetchClientCodegen;

import java.util.*;

public class TypeScriptFetchGlobalsClientCodegen extends TypeScriptFetchClientCodegen {

    public TypeScriptFetchGlobalsClientCodegen() {
        super();

        outputFolder = "generated-code/typescript-fetch-globals";
        embeddedTemplateDir = templateDir = "typescript-fetch-globals";

    }

    @Override
    public void processOpts() {
        super.processOpts();
        // remove files we definitely don't use
        supportingFiles.removeIf(f -> f.templateFile == "custom.d.mustache");
        supportingFiles.removeIf(f -> f.templateFile == "git_push.sh.mustache");
        supportingFiles.removeIf(f -> f.templateFile == "gitignore");
    }

    @Override
    public String getName() {
        return "typescript-fetch-globals";
    }

    @Override
    public String getHelp() {
        return "Generates a TypeScript client library using Fetch API (beta).";
    }
}
