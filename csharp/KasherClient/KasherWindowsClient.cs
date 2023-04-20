using System.Net;
using System.Net.Sockets;
using System.Reflection;
using System.Text;

namespace Kasher;

internal static class KasherWindowsClient
{
    private record KasherArgs
    {
        public ushort LocalPort { get; init; }
        public string ServerHost { get; init; } = null!;
        public string Destination { get; init; } = null!;
    }
    
    private static readonly HttpClient Client = new(
        new HttpClientHandler
        {
            ServerCertificateCustomValidationCallback = HttpClientHandler.DangerousAcceptAnyServerCertificateValidator
        });

    private static async Task DoTheFunction(Socket client, KasherArgs args)
    {
        var buffer = new byte[1024 * 640];
        var hostUrl = $"{args.ServerHost}/{Guid.NewGuid()}";
        var content = new ByteArrayContent(Encoding.ASCII.GetBytes(args.Destination));
        await Client.PostAsync(hostUrl, content);

        var socketStream = new NetworkStream(client);
        var _ = Task.Run(async () =>
        {
            while (client.Connected)
            {
                var stream = await Client.GetStreamAsync(hostUrl);
                await stream.CopyToAsync(socketStream);
            }
        });
        while (client.Connected)
        {
            var length = await socketStream.ReadAsync(buffer);
            content = new ByteArrayContent(buffer[..length]);
            await Client.PutAsync(hostUrl, content);
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
    
    public static async Task<int> Main(string[] args)
    {
        var kasherArgs = ParseArgs(args);
        var server = new TcpListener(IPAddress.Loopback, kasherArgs.LocalPort);
        server.Start();
        while (true)
        {
            var base64Method = Convert.FromBase64String("QWNjZXB0U29ja2V0QXN5bmM="); // AcceptSocketAsync
            var decoded = Encoding.UTF8.GetString(base64Method);
            var acceptMethod = typeof(TcpListener).GetMethod(decoded, Type.EmptyTypes);
            var client = await (Task<Socket>) acceptMethod.Invoke(server, null);
            Console.WriteLine($"Taking new client");
            client.SetSocketOption(SocketOptionLevel.Socket, SocketOptionName.KeepAlive, true);
            var _ = Task.Run(() => DoTheFunction(client, kasherArgs));
        }
    }
}