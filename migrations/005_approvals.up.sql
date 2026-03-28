-- Time entry approvals audit trail
CREATE TABLE time_entry_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    time_entry_id UUID NOT NULL REFERENCES time_entries(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL CHECK (action IN ('submit', 'approve', 'reject', 'edit_approve', 'edit_return', 'partial_approve', 'delegate')),
    actor_user_id UUID NOT NULL REFERENCES users(id),
    actor_role VARCHAR(50) NOT NULL,
    changes JSONB,
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Expense approvals audit trail
CREATE TABLE expense_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expense_id UUID NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL CHECK (action IN ('submit', 'approve', 'reject', 'edit_approve', 'edit_return', 'partial_approve', 'delegate')),
    actor_user_id UUID NOT NULL REFERENCES users(id),
    actor_role VARCHAR(50) NOT NULL,
    changes JSONB,
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Designated backup approvers per organization per role
CREATE TABLE backup_approvers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL CHECK (role IN ('manager', 'finance')),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(organization_id, role, user_id)
);

-- Indexes for performance
CREATE INDEX idx_time_entry_approvals_entry_id ON time_entry_approvals(time_entry_id);
CREATE INDEX idx_time_entry_approvals_actor_id ON time_entry_approvals(actor_user_id);
CREATE INDEX idx_time_entry_approvals_created_at ON time_entry_approvals(created_at);

CREATE INDEX idx_expense_approvals_expense_id ON expense_approvals(expense_id);
CREATE INDEX idx_expense_approvals_actor_id ON expense_approvals(actor_user_id);
CREATE INDEX idx_expense_approvals_created_at ON expense_approvals(created_at);

CREATE INDEX idx_backup_approvers_org_role ON backup_approvers(organization_id, role);
CREATE INDEX idx_backup_approvers_user_id ON backup_approvers(user_id);
