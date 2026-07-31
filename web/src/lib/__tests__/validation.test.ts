import { describe, it, expect } from "vitest";
import { z } from "zod";

// ── Exported schemas from @/types/unit ──────────────────────────────────
import {
  CreateUnitRequestSchema,
  UpdateUnitRequestSchema,
  AddUnitMemberRequestSchema,
} from "@/types/unit";

// ── Exported schemas from route components (recreated inline for isolation) ──
const tabsSchema = z.enum(["owned", "adopted", "all"]);
const statusSchema = z.enum(["all", "active", "inactive"]);

// ── Auth form schemas (recreated from route components) ───────────────────
const loginSchema = z.object({
  identifier: z.string().min(1, "Username or email is required"),
  password: z.string().min(1, "Password is required"),
});

const bootstrapSchema = z.object({
  organization_name: z.string().min(1, "Organization name is required"),
  email: z.string().email("Invalid email address"),
  username: z.string().min(3, "Username must be at least 3 characters"),
  firstname: z.string().min(1, "First name is required"),
  lastname: z.string().min(1, "Last name is required"),
  password: z.string().min(8, "Password must be at least 8 characters"),
});

// ── Customer form schema (recreated from route component) ─────────────────
const customerFormSchema = z.object({
  company_name: z.string().min(1, "Company name is required"),
  contact_name: z.string().optional(),
  email: z.string().email("Invalid email").optional().or(z.literal("")),
  phone: z.string().optional(),
  vat_number: z.string().optional(),
  address: z.string().optional(),
});

// ── Time entries route search schema ──────────────────────────────────────
const timeEntrySearchSchema = z.object({
  date: z.coerce.date().default(new Date()),
  month: z.coerce.date().default(new Date()),
});

// ── Tests ────────────────────────────────────────────────────────────────

describe("Zod validation schemas", () => {
  describe("Unit schemas (exported from @/types/unit)", () => {
    it("CreateUnitRequestSchema accepts valid data", () => {
      const valid = {
        name: "Engineering",
        description: null,
        parent_unit_id: null,
        code: "ENG",
      };
      expect(CreateUnitRequestSchema.safeParse(valid).success).toBe(true);
    });

    it("CreateUnitRequestSchema rejects empty name", () => {
      const invalid = {
        name: "",
        description: null,
        parent_unit_id: null,
        code: "ENG",
      };
      expect(CreateUnitRequestSchema.safeParse(invalid).success).toBe(false);
    });

    it("CreateUnitRequestSchema rejects code > 50 chars", () => {
      const invalid = {
        name: "Test",
        description: null,
        parent_unit_id: null,
        code: "A".repeat(51),
      };
      expect(CreateUnitRequestSchema.safeParse(invalid).success).toBe(false);
    });

    it("UpdateUnitRequestSchema accepts valid partial update", () => {
      const valid = {
        name: "Updated Unit",
        description: "New desc",
        code: "UPD",
      };
      expect(UpdateUnitRequestSchema.safeParse(valid).success).toBe(true);
    });

    it("AddUnitMemberRequestSchema accepts valid member data", () => {
      const valid = {
        user_id: "550e8400-e29b-41d4-a716-446655440000",
        role: "Viewer",
        is_primary: false,
      };
      expect(AddUnitMemberRequestSchema.safeParse(valid).success).toBe(true);
    });

    it("AddUnitMemberRequestSchema rejects non-uuid user_id", () => {
      const invalid = {
        user_id: "not-a-uuid",
        role: "Viewer",
        is_primary: false,
      };
      expect(AddUnitMemberRequestSchema.safeParse(invalid).success).toBe(false);
    });
  });

  describe("Contract list schemas (route search params)", () => {
    it("tabsSchema accepts owned, adopted, all", () => {
      expect(tabsSchema.safeParse("owned").success).toBe(true);
      expect(tabsSchema.safeParse("adopted").success).toBe(true);
      expect(tabsSchema.safeParse("all").success).toBe(true);
    });

    it("tabsSchema rejects invalid tab value", () => {
      expect(tabsSchema.safeParse("invalid").success).toBe(false);
      expect(tabsSchema.safeParse("").success).toBe(false);
    });

    it("statusSchema accepts all, active, inactive", () => {
      expect(statusSchema.safeParse("all").success).toBe(true);
      expect(statusSchema.safeParse("active").success).toBe(true);
      expect(statusSchema.safeParse("inactive").success).toBe(true);
    });
  });

  describe("Login form schema", () => {
    it("accepts valid identifier and password", () => {
      expect(
        loginSchema.safeParse({
          identifier: "user@test.com",
          password: "password123",
        }).success
      ).toBe(true);
      expect(
        loginSchema.safeParse({ identifier: "username", password: "p" }).success
      ).toBe(true);
    });

    it("rejects missing identifier", () => {
      expect(
        loginSchema.safeParse({ identifier: "", password: "password123" })
          .success
      ).toBe(false);
    });

    it("rejects missing password", () => {
      expect(
        loginSchema.safeParse({ identifier: "user", password: "" }).success
      ).toBe(false);
    });
  });

  describe("Bootstrap form schema", () => {
    it("accepts valid bootstrap data", () => {
      const valid = {
        organization_name: "My Org",
        email: "admin@example.com",
        username: "admin",
        firstname: "Admin",
        lastname: "User",
        password: "securepass123",
      };
      expect(bootstrapSchema.safeParse(valid).success).toBe(true);
    });

    it("rejects invalid email", () => {
      const invalid = {
        organization_name: "My Org",
        email: "not-an-email",
        username: "admin",
        firstname: "Admin",
        lastname: "User",
        password: "securepass123",
      };
      expect(bootstrapSchema.safeParse(invalid).success).toBe(false);
    });

    it("rejects short password", () => {
      const invalid = {
        organization_name: "My Org",
        email: "admin@example.com",
        username: "admin",
        firstname: "Admin",
        lastname: "User",
        password: "short",
      };
      expect(bootstrapSchema.safeParse(invalid).success).toBe(false);
    });

    it("rejects short username", () => {
      const invalid = {
        organization_name: "My Org",
        email: "admin@example.com",
        username: "ab",
        firstname: "Admin",
        lastname: "User",
        password: "securepass123",
      };
      expect(bootstrapSchema.safeParse(invalid).success).toBe(false);
    });
  });

  describe("Customer form schema", () => {
    it("accepts valid customer with all fields", () => {
      const valid = {
        company_name: "Acme Corp",
        contact_name: "John Doe",
        email: "john@acme.com",
        phone: "+1 234 567",
        vat_number: "VAT123",
        address: "123 Main St",
      };
      expect(customerFormSchema.safeParse(valid).success).toBe(true);
    });

    it("accepts customer with only required fields", () => {
      const valid = { company_name: "Acme Corp" };
      expect(customerFormSchema.safeParse(valid).success).toBe(true);
    });

    it("rejects empty company_name", () => {
      const invalid = { company_name: "" };
      expect(customerFormSchema.safeParse(invalid).success).toBe(false);
    });

    it("rejects invalid email when provided", () => {
      const invalid = { company_name: "Acme", email: "not-email" };
      expect(customerFormSchema.safeParse(invalid).success).toBe(false);
    });

    it("accepts empty string email", () => {
      const valid = { company_name: "Acme", email: "" };
      expect(customerFormSchema.safeParse(valid).success).toBe(true);
    });
  });

  describe("Time entry route search schema", () => {
    it("coerces string date to Date", () => {
      const result = timeEntrySearchSchema.safeParse({
        date: "2026-05-18",
        month: "2026-05-01",
      });
      expect(result.success).toBe(true);
      if (result.success) {
        expect(result.data.date).toBeInstanceOf(Date);
        expect(result.data.month).toBeInstanceOf(Date);
      }
    });
  });
});
