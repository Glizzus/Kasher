// See https://aka.ms/new-console-template for more information

using System.Net;

class KasherServer
{

    private static HttpListener _listener;

    public static void Main(string[] args)
    {
        _listener = new();
        _listener.Prefixes.Add($"https://localhost:{args[0]}");
        _listener.Start();
        
        H
    }

}

HttpListener listener = new();
listener.Prefixes.Add("https://localhost");