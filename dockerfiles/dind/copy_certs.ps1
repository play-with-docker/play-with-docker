param (
        [Parameter(Mandatory = $true)]
        [string] $Node,
        [Parameter(Mandatory = $true)]
        [string] $SessionId,
        [Parameter(Mandatory = $true)]
        [string] $FQDN
)


function GetDirectUrlFromIp ($ip) {
        $ip_dash=$ip -replace "\.","-"
        $url="https://ip${ip_dash}-${SessionId}.direct.${FQDN}"
        return $url
}

function WaitForUrl ($url) {
    write-host $url
        do {
                try{
            invoke-webrequest -UseBasicParsing -uri $url | Out-Null
        } catch {}
        $status = $?
        sleep 1
        } until($status)
}

function GetNodeRoutableIp ($nodeName) {
  $JQFilter='.instances[] | select (.hostname == \"{0}\") | .routable_ip' -f $nodeName
  $rip = (invoke-webrequest -UseBasicParsing -uri "https://$FQDN/sessions/$SessionId").Content |  jq -r $JQFilter

  IF([string]::IsNullOrEmpty($rip)) {
    Write-Host "Could not fetch IP for node $nodeName"
    exit 1
  }
  return $rip
}

function Set-UseUnsafeHeaderParsing
{
    param(
        [Parameter(Mandatory,ParameterSetName='Enable')]
        [switch]$Enable,

        [Parameter(Mandatory,ParameterSetName='Disable')]
        [switch]$Disable
    )

    $ShouldEnable = $PSCmdlet.ParameterSetName -eq 'Enable'

    $netAssembly = [Reflection.Assembly]::GetAssembly([System.Net.Configuration.SettingsSection])

    if($netAssembly)
    {
        $bindingFlags = [Reflection.BindingFlags] 'Static,GetProperty,NonPublic'
        $settingsType = $netAssembly.GetType('System.Net.Configuration.SettingsSectionInternal')

        $instance = $settingsType.InvokeMember('Section', $bindingFlags, $null, $null, @())

        if($instance)
        {
            $bindingFlags = 'NonPublic','Instance'
            $useUnsafeHeaderParsingField = $settingsType.GetField('useUnsafeHeaderParsing', $bindingFlags)

            if($useUnsafeHeaderParsingField)
            {
              $useUnsafeHeaderParsingField.SetValue($instance, $ShouldEnable)
            }
        }
    }
}


$ProgressPreference = 'SilentlyContinue'
$ErrorActionPreference = 'Stop'

Set-UseUnsafeHeaderParsing -Enable

Start-Transcript -path ("C:\{0}.log" -f $MyInvocation.MyCommand.Name) -append

add-type @"
    using System.Net;
    using System.Security.Cryptography.X509Certificates;

    public class IDontCarePolicy : ICertificatePolicy {
        public IDontCarePolicy() {}
        public bool CheckValidationResult(
            ServicePoint sPoint, X509Certificate cert,
            WebRequest wRequest, int certProb) {
            return true;
        }
    }
"@

[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

[System.Net.ServicePointManager]::CertificatePolicy = new-object IDontCarePolicy


$dtr_ip = GetNodeRoutableIp $Node
$dtr_url = GetDirectUrlFromIp $dtr_ip
$dtr_hostname = $dtr_url -replace "https://",""

WaitForUrl "${dtr_url}/ca"

invoke-webrequest -UseBasicParsing -uri "$dtr_url/ca" -o c:\ca.crt

$cert = new-object System.Security.Cryptography.X509Certificates.X509Certificate2 c:\ca.crt
$store = new-object System.Security.Cryptography.X509Certificates.X509Store('Root','localmachine')
$store.Open('ReadWrite')
$store.Add($cert)
$store.Close()

Stop-Transcript
