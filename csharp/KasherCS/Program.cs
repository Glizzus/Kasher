using System.Net;
using System.Net.Sockets;
using System.Text;

class KasherWindowsClient
{
    private record KasherArgs
    {
        public ushort LocalPort { get; init; }
        public string ServerHost { get; init; }
        public string Destination { get; init; }
    }
    
    private static HttpClient _client = new(
        new HttpClientHandler()
        {
            ServerCertificateCustomValidationCallback = HttpClientHandler.DangerousAcceptAnyServerCertificateValidator
        });

    static async Task HandleConnection(Socket client, KasherArgs args)
    {
        var buffer = new byte[1024 * 640];
        var hostUrl = $"{args.ServerHost}/{Guid.NewGuid()}";
        var content = new ByteArrayContent(Encoding.ASCII.GetBytes(args.Destination));
        await _client.PostAsync(hostUrl, content);

        var socketStream = new NetworkStream(client);
        var _ = Task.Run(async () =>
        {
            while (client.Connected)
            {
                Thread.Sleep(25);
                var stream = await _client.GetStreamAsync(hostUrl);
                await stream.CopyToAsync(socketStream);
            }
        });
        while (client.Connected)
        {
            var length = await socketStream.ReadAsync(buffer);
            content = new ByteArrayContent(buffer[..length]);
            await _client.PutAsync(hostUrl, content);
        }
    }

    private static KasherArgs ParseArgs(IReadOnlyList<string> args)
    {
        if (args.Count < 3)
        {
            throw new InvalidDataException("Not enough args, 3 required");
        }

        if (!ushort.TryParse(args[0], out var port))
        {
            throw new InvalidDataException("First argument is not a valid port");
        }
        var server = args[1];
        var destination = args[2];

        return new KasherArgs
        {
            LocalPort = port,
            ServerHost = server,
            Destination = destination
        };
    }
    
    public static async Task Main(string[] args)
    {
        var kasherArgs = ParseArgs(args);
        var server = new TcpListener(IPAddress.Loopback, kasherArgs.LocalPort);
        server.Start();
        while (true)
        {
            var client = await server.AcceptSocketAsync();
            Console.WriteLine($"Taking new client");
            client.SetSocketOption(SocketOptionLevel.Socket, SocketOptionName.KeepAlive, true);
            var _ = Task.Run(() => HandleConnection(client, kasherArgs));
        }
    }
}