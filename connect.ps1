Param(
	[Parameter(Mandatory=$True)]
	[string]$endpoint
)

function RegisterEvent {
    if ($event) {
        Unregister-Event $event
    }
    ($global:event = Register-ObjectEvent -InputObject $session.Runspace -EventName AvailabilityChanged -Action {
        if ($session.State -eq "Broken") {
            $global:session = New-PSSession -HostName $endpoint -UserName Administrator
            RegisterEvent
        } 
        if ($Host.Runspace -ne $session.Runspace) {
            Enter-PSSession $session
        }
    }) | Out-Null
}


$global:session = New-PSSession -HostName $endpoint -UserName Administrator
Enter-PSSession $session 
RegisterEvent


