import { WebSocketGateway, WebSocketServer } from '@nestjs/websockets';
import { Server } from 'socket.io';

// Live feed of posted entries. Clients connect to the /events namespace and
// listen for "entry.posted". Replays are not rebroadcast: the entry already
// went out on its first posting.
@WebSocketGateway({ namespace: 'events', cors: true })
export class EventsGateway {
  @WebSocketServer()
  server!: Server;

  broadcast(event: string, payload: unknown) {
    this.server?.emit(event, payload);
  }
}
