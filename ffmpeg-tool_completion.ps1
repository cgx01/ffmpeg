$scriptBlock = {
    param($wordToComplete, $commandAst, $cursorPosition)

    $commandElements = $commandAst.CommandElements
    $commandName = $commandElements[0].Value
    $subCommand = $null

    # 1. Determine sub-command
    if ($commandElements.Count -gt 1) {
        $possibleSubCommand = $commandElements[1].Value
        if ($possibleSubCommand -in 'convert', 'unzip', 'gif', 'flatten', 'stats', 'help') {
            $subCommand = $possibleSubCommand
        }
    }

    # 2. Sub-command completion
    if ($null -eq $subCommand) {
        $subcommands = 'convert', 'unzip', 'gif', 'flatten', 'stats', 'help'
        $subcommands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "SubCommand: $_")
        }
        return
    }

    # 3. Flag completion
    $flags = @()
    switch ($subCommand) {
        'convert' {
            $flags = @('--dirs', '-d', '--to', '--vcodec', '--accel', '--workers', '--force', '--dry-run')
        }
        'unzip' {
            $flags = @('--dir', '-d', '--password', '-p')
        }
        'gif' {
            $flags = @('--dir', '-d', '--size', '-s')
        }
        'flatten' {
            $flags = @('--dir', '-d')
        }
        'stats' {
            $flags = @('--dirs', '-d')
        }
    }
    
    # Global flags
    $flags += @('--ffmpeg')

    # Output matching flags
    $flags | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Flag $_")
    }
}

Register-ArgumentCompleter -Native -CommandName ffmpeg-tool -ScriptBlock $scriptBlock
