Register-ArgumentCompleter -Native -CommandName ffmpeg-tool -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $commandElements = $commandAst.CommandElements
    $commandName = $commandElements[0].Value
    $subCommand = $null

    # 1. 尝试识别当前的子命令
    if ($commandElements.Count -gt 1) {
        $possibleSubCommand = $commandElements[1].Value
        if ($possibleSubCommand -in 'convert', 'unzip', 'gif', 'flatten', 'stats', 'help') {
            $subCommand = $possibleSubCommand
        }
    }

    # 2. 如果还没有子命令，或者光标在第一个参数位置，提供子命令补全
    # 注意：$commandElements.Count 在输入第一个空格后会增加
    if ($null -eq $subCommand) {
        $subcommands = 'convert', 'unzip', 'gif', 'flatten', 'stats', 'help'
        return $subcommands | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "SubCommand: $_")
        }
    }

    # 3. 根据子命令提供特定的 Flag 补全
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
    
    # 加上全局 Flag
    $flags += @('--ffmpeg')

    # 过滤并返回匹配的 Flag
    return $flags | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', "Flag $_")
    }
}
