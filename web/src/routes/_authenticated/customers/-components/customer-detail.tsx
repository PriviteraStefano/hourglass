import { useNavigate } from "@tanstack/react-router";
import { useSuspenseQuery } from "@tanstack/react-query";
import { ArrowLeft, Mail, MapPin, Phone } from "lucide-react";
import { Button } from "@/components/ui/button.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card.tsx";
import { CustomersApis } from "@/api/customers";
import { AuthApis } from "@/api/auth";
import { Route } from "@/routes/_authenticated/customers/$id";

export function CustomerDetail() {
  const { id } = Route.useParams();
  const navigate = useNavigate();
  const {
    data: { customer, linked_contracts },
  } = useSuspenseQuery(CustomersApis.customerQueryOpts(id));
  const { data: me } = useSuspenseQuery(AuthApis.profileQueryOpts);

  const isFinance = me?.membership?.role === "finance";

  return (
    <div className="space-y-4">
      <Button
        variant="ghost"
        size="sm"
        onClick={() => navigate({ to: "/customers" })}
      >
        <ArrowLeft className="w-4 h-4 mr-1" />
        Back to Customers
      </Button>

      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-semibold">{customer.company_name}</h1>
          {customer.is_internal && <Badge variant="secondary">Internal</Badge>}
          <Badge
            variant={customer.is_active ? "default" : "outline"}
            className={customer.is_active ? "bg-green-500" : ""}
          >
            {customer.is_active ? "Active" : "Inactive"}
          </Badge>
        </div>
        {isFinance && (
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => navigate({ to: "/customers" })}
            >
              Edit
            </Button>
            <Button
              variant="outline"
              className="text-destructive hover:text-destructive"
              onClick={() => navigate({ to: "/customers" })}
            >
              Delete
            </Button>
          </div>
        )}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Customer Information</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {customer.contact_name && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">Contact Name</span>
              <span>{customer.contact_name}</span>
            </div>
          )}
          {customer.email && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">Email</span>
              <div className="flex items-center gap-1">
                <Mail className="h-3.5 w-3.5 text-muted-foreground" />
                <span>{customer.email}</span>
              </div>
            </div>
          )}
          {customer.phone && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">Phone</span>
              <div className="flex items-center gap-1">
                <Phone className="h-3.5 w-3.5 text-muted-foreground" />
                <span>{customer.phone}</span>
              </div>
            </div>
          )}
          {customer.vat_number && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">VAT Number</span>
              <span>{customer.vat_number}</span>
            </div>
          )}
          {customer.address && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">Address</span>
              <div className="flex items-center gap-1">
                <MapPin className="h-3.5 w-3.5 text-muted-foreground" />
                <span>{customer.address}</span>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Contracts</CardTitle>
        </CardHeader>
        <CardContent>
          {linked_contracts && linked_contracts.length > 0 ? (
            <div className="divide-y">
              {linked_contracts.map((contract) => (
                <div
                  key={contract.id}
                  className="py-3 flex items-center justify-between"
                >
                  <div>
                    <div className="font-medium">{contract.name}</div>
                    <div className="text-sm text-muted-foreground capitalize">
                      {contract.governance_model.replace("_", " ")}
                    </div>
                  </div>
                  <Badge
                    variant={contract.is_active ? "default" : "outline"}
                    className={contract.is_active ? "bg-green-500" : ""}
                  >
                    {contract.is_active ? "Active" : "Inactive"}
                  </Badge>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-4 text-muted-foreground">
              No linked contracts
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
