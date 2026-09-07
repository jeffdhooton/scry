export interface Speaker {
  speak(): string;
}

export class Greeter implements Speaker {
  speak(): string { return "hello"; }
}

export function invoke(s: Speaker): string { return s.speak(); }

export function unrelated(): number { return 7; }
