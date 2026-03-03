import { gql } from "urql";

export interface StartRunInput {
  clientRequestId: string;
  roomSlug: string;
}

export interface StartRunPayload {
  runId: string;
  runToken: string;
}

export interface StartRunResult {
  startRun: StartRunPayload;
}

export const START_RUN_MUTATION = gql`
  mutation StartRun($input: StartRunInput!) {
    startRun(input: $input) {
      runId
      runToken
    }
  }
`;
