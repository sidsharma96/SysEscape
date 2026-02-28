import { gql } from "urql";

export interface RoomVersion {
  id: string;
  versionNumber: number;
  status: "PUBLISHED" | "DEPRECATED" | "DISABLED";
  changelog: string;
  publishedAt: string;
}

export interface Room {
  id: string;
  slug: string;
  title: string;
  district: string;
  engine: "A" | "B";
  difficulty: "L0" | "L1" | "L2" | "L3";
  description: string;
  latestVersion: RoomVersion | null;
}

export interface RoomsQueryResult {
  rooms: Room[];
}

export interface RoomBySlugQueryResult {
  roomBySlug: Room | null;
}

export const ROOMS_QUERY = gql`
  query Rooms($engine: RoomEngine, $difficulty: RoomDifficulty, $district: String) {
    rooms(engine: $engine, difficulty: $difficulty, district: $district) {
      slug
      title
      district
      engine
      difficulty
      description
      latestVersion {
        id
        versionNumber
        status
        changelog
        publishedAt
      }
    }
  }
`;

export const ROOM_BY_SLUG_QUERY = gql`
  query RoomBySlug($slug: String!) {
    roomBySlug(slug: $slug) {
      slug
      title
      district
      engine
      difficulty
      description
      latestVersion {
        id
        versionNumber
        status
        changelog
        publishedAt
      }
    }
  }
`;
