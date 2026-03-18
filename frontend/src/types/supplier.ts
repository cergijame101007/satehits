/** お取り引き先 */
export interface Supplier {
  id: number;
  name: string;
  description: string;
  instagram_url: string | null;
  image_url: string | null;
  display_order: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}
