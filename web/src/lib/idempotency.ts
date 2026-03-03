import { v4 as uuidv4 } from "uuid";

export function newRequestId(): string {
  return uuidv4();
}
