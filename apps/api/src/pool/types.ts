export interface PoolMessage {
  type: string;
}

export interface WelcomeMessage extends PoolMessage {
  type: "welcome";
  epoch: string;
  app_id: string;
}

export interface EventMessage extends PoolMessage {
  type: "event";
  id: string;
  channel: string;
  offset: number;
  payload: unknown;
  timestamp: number;
  sender: string;
}

export interface AckMessage extends PoolMessage {
  type: "ack";
  channel: string;
  offset: number;
}

export interface SubscribeMessage extends PoolMessage {
  type: "subscribe";
  channels: Record<string, { last_offset: number }>;
}

export interface ErrorMessage extends PoolMessage {
  type: "error";
  code: string;
  message: string;
}

export interface RegistrationRequest {
  app: string;
  secret: string;
  events?: {
    emit?: string[];
    listen?: string[];
  };
}

export interface RegisteredApp {
  app_id: string;
  app: string;
  token: string;
  emit_channels: string[];
  listen_channels: string[];
  registered_at: string;
}

export interface ConnectedApp {
  app_id: string;
  app: string;
  ws: unknown;
  channels: Set<string>;
}
